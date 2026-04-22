package handlers

import (
	"database/sql"
	"docuflow/models"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type DocumentHandler struct {
	DB *sql.DB
}

func (h *DocumentHandler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SELECT id, title, updated_at, COALESCE(share_token,''), COALESCE(share_password,'') FROM documents ORDER BY updated_at DESC")
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type DocItem struct {
		models.Document
		IsProtected bool
		IsShared    bool
	}

	var docs []DocItem
	for rows.Next() {
		var d models.Document
		if err := rows.Scan(&d.ID, &d.Title, &d.UpdatedAt, &d.ShareToken, &d.SharePassword); err != nil {
			continue
		}
		docs = append(docs, DocItem{
			Document:    d,
			IsProtected: d.SharePassword != "",
			IsShared:    d.ShareToken != "",
		})
	}

	tmpl := template.Must(template.ParseFiles("web/templates/base.html", "web/templates/document_list.html"))
	tmpl.Execute(w, struct {
		User      string
		Documents []DocItem
	}{
		User:      GetBaseData(r).User,
		Documents: docs,
	})
}

func (h *DocumentHandler) NewDocument(w http.ResponseWriter, r *http.Request) {
	folderID := r.URL.Query().Get("folder_id")

	if r.Method == "GET" {
		tmpl := template.Must(template.ParseFiles("web/templates/base.html", "web/templates/document_edit.html"))
		
		var folderIDPtr *int
		if folderID != "" {
			// Basic validation/parsing could go here
		}

		tmpl.Execute(w, struct {
			User     string
			FolderID string
			models.Document
		}{
			User:     GetBaseData(r).User,
			FolderID: folderID,
		})
		return
	}

	title := r.FormValue("title")
	if title == "" {
		title = "Untitled Document"
	}
	content := r.FormValue("content")
	folderIDPost := r.FormValue("folder_id")
	ownerID := 1

	var res sql.Result
	var err error
	if folderIDPost == "" {
		res, err = h.DB.Exec("INSERT INTO documents (title, content, owner_id) VALUES (?, ?, ?)", title, content, ownerID)
	} else {
		res, err = h.DB.Exec("INSERT INTO documents (title, content, owner_id, folder_id) VALUES (?, ?, ?, ?)", title, content, ownerID, folderIDPost)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := res.LastInsertId()
	http.Redirect(w, r, fmt.Sprintf("/documents/view?id=%d", id), http.StatusSeeOther)
}

func (h *DocumentHandler) ViewDocument(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	var doc models.Document
	err := h.DB.QueryRow(
		"SELECT id, title, content, COALESCE(share_token,''), COALESCE(share_password,'') FROM documents WHERE id = ?", id,
	).Scan(&doc.ID, &doc.Title, &doc.Content, &doc.ShareToken, &doc.SharePassword)
	if err != nil {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}

	// Render Markdown
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	htmlBytes := markdown.ToHTML([]byte(doc.Content), p, nil)

	// Load attached files
	files, _ := GetDocumentFiles(h.DB, id)

	shareURL := ""
	if doc.ShareToken != "" {
		shareURL = fmt.Sprintf("http://%s/share/%s", r.Host, doc.ShareToken)
	}

	tmpl := template.Must(template.ParseFiles("web/templates/base.html", "web/templates/document_view.html"))
	tmpl.Execute(w, struct {
		User        string
		Document    models.Document
		Content     template.HTML
		Files       []map[string]interface{}
		ShareURL    string
		IsProtected bool
	}{
		User:        GetBaseData(r).User,
		Document:    doc,
		Content:     template.HTML(htmlBytes),
		Files:       files,
		ShareURL:    shareURL,
		IsProtected: doc.SharePassword != "",
	})
}

func (h *DocumentHandler) EditDocument(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	if r.Method == "GET" {
		var doc models.Document
		err := h.DB.QueryRow("SELECT id, title, content FROM documents WHERE id = ?", id).Scan(&doc.ID, &doc.Title, &doc.Content)
		if err != nil {
			http.Error(w, "Document not found", http.StatusNotFound)
			return
		}
		tmpl := template.Must(template.ParseFiles("web/templates/base.html", "web/templates/document_edit.html"))
		tmpl.Execute(w, struct {
			User string
			models.Document
		}{
			User:     GetBaseData(r).User,
			Document: doc,
		})
		return
	}

	// POST: Update document
	title := r.FormValue("title")
	content := r.FormValue("content")

	var oldContent string
	h.DB.QueryRow("SELECT content FROM documents WHERE id = ?", id).Scan(&oldContent)

	h.DB.Exec("INSERT INTO revisions (document_id, content, editor_id, change_summary) VALUES (?, ?, ?, ?)",
		id, oldContent, 1, "Manual save")

	_, err := h.DB.Exec("UPDATE documents SET title = ?, content = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", title, content, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/documents/view?id="+id, http.StatusSeeOther)
}

// Autosave endpoint for HTMX
func (h *DocumentHandler) Autosave(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.FormValue("id")
	content := r.FormValue("content")

	_, err := h.DB.Exec("UPDATE documents SET content = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", content, id)
	if err != nil {
		http.Error(w, "Save failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<span style="color: #22c55e;">Saved</span>`))
}

// SetPassword sets or clears the share password for a document.
func (h *DocumentHandler) SetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.FormValue("id")
	password := r.FormValue("password")

	if strings.TrimSpace(password) == "" {
		// Clear password
		h.DB.Exec("UPDATE documents SET share_password = '' WHERE id = ?", id)
	} else {
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		h.DB.Exec("UPDATE documents SET share_password = ? WHERE id = ?", string(hashed), id)
	}

	http.Redirect(w, r, "/documents/view?id="+id, http.StatusSeeOther)
}

// GenerateShareLink creates or returns a unique share token for a document.
func (h *DocumentHandler) GenerateShareLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.FormValue("id")
	action := r.FormValue("action") // "generate" or "revoke"

	if action == "revoke" {
		h.DB.Exec("UPDATE documents SET share_token = NULL WHERE id = ?", id)
	} else {
		token := uuid.New().String()
		h.DB.Exec("UPDATE documents SET share_token = ? WHERE id = ?", token, id)
	}

	http.Redirect(w, r, "/documents/view?id="+id, http.StatusSeeOther)
}

// ShareView handles the public share link, with optional password gate.
func (h *DocumentHandler) ShareView(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.URL.Path, "/share/")
	if token == "" {
		http.Error(w, "Invalid share link", http.StatusBadRequest)
		return
	}

	var doc models.Document
	err := h.DB.QueryRow(
		"SELECT id, title, content, COALESCE(share_password,'') FROM documents WHERE share_token = ?", token,
	).Scan(&doc.ID, &doc.Title, &doc.Content, &doc.SharePassword)
	if err != nil {
		http.Error(w, "This link is invalid or has been revoked.", http.StatusNotFound)
		return
	}

	isProtected := doc.SharePassword != ""

	// Password gate check
	if isProtected {
		if r.Method == "POST" {
			r.ParseForm()
			attempt := r.FormValue("share_password")
			if bcrypt.CompareHashAndPassword([]byte(doc.SharePassword), []byte(attempt)) != nil {
				// Wrong password — re-show gate with error
				tmpl := template.Must(template.ParseFiles("web/templates/base.html", "web/templates/share_view.html"))
				tmpl.Execute(w, struct {
					Title       string
					ShowGate    bool
					GateError   string
					Token       string
				}{
					Title:     doc.Title,
					ShowGate:  true,
					GateError: "Incorrect password. Please try again.",
					Token:     token,
				})
				return
			}
			// Correct — set a session cookie for this share token and reload
			http.SetCookie(w, &http.Cookie{
				Name:     "share_" + token,
				Value:    "1",
				Expires:  time.Now().Add(4 * time.Hour),
				HttpOnly: true,
			})
			http.Redirect(w, r, "/share/"+token, http.StatusSeeOther)
			return
		}

		// GET: check if already unlocked via cookie
		if _, err := r.Cookie("share_" + token); err != nil {
			// Show password gate
			tmpl := template.Must(template.ParseFiles("web/templates/base.html", "web/templates/share_view.html"))
			tmpl.Execute(w, struct {
				Title     string
				ShowGate  bool
				GateError string
				Token     string
			}{
				Title:    doc.Title,
				ShowGate: true,
				Token:    token,
			})
			return
		}
	}

	// Render document content
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	htmlBytes := markdown.ToHTML([]byte(doc.Content), p, nil)

	files, _ := GetDocumentFiles(h.DB, fmt.Sprintf("%d", doc.ID))

	tmpl := template.Must(template.ParseFiles("web/templates/base.html", "web/templates/share_view.html"))
	tmpl.Execute(w, struct {
		Title    string
		ShowGate bool
		Document models.Document
		Content  template.HTML
		Files    []map[string]interface{}
		Token    string
	}{
		Title:    doc.Title,
		ShowGate: false,
		Document: doc,
		Content:  template.HTML(htmlBytes),
		Files:    files,
		Token:    token,
	})
}
