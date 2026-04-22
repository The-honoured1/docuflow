package handlers

import (
	"database/sql"
	"docuflow/models"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)


var folderIDPtr *int

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

	data := GetBaseData(r)
	data.Documents = docs
	data.View = "document_list"

	tmpl := template.Must(template.ParseFiles("web/templates/index.html"))
	tmpl.Execute(w, data)
}


func (h *DocumentHandler) NewDocument(w http.ResponseWriter, r *http.Request) {
	folderID := r.URL.Query().Get("folder_id")

	if r.Method == "GET" {
		tmpl := template.Must(template.ParseFiles("web/templates/index.html"))

		if folderID != "" {
			// Basic validation/parsing could go here
		}

		data := GetBaseData(r)
		data.FolderID = folderID
		data.View = "document_new"

		tmpl.Execute(w, data)
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

	// Load Snapshots
	rows, _ := h.DB.Query(`SELECT id, created_at FROM revisions WHERE document_id = ? ORDER BY created_at DESC LIMIT 10`, id)
	var revisions []models.Revision
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var r models.Revision
			rows.Scan(&r.ID, &r.CreatedAt)
			revisions = append(revisions, r)
		}
	}

	files, _ := GetDocumentFiles(h.DB, id)
	shareURL := ""
	if doc.ShareToken != "" {
		shareURL = fmt.Sprintf("http://%s/share/%s", r.Host, doc.ShareToken)
	}

	data := GetBaseData(r)
	data.Document = doc
	data.Files = files
	data.ShareURL = shareURL
	data.IsProtected = doc.SharePassword != ""
	data.View = "document_view"
	data.RevisionsList = revisions

	tmpl := template.Must(template.ParseFiles("web/templates/index.html"))
	tmpl.Execute(w, data)
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
		data := GetBaseData(r)
		data.Document = doc
		data.View = "document_edit"

		tmpl := template.Must(template.ParseFiles("web/templates/index.html"))
		tmpl.Execute(w, data)
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

	// Automatic Versioning: Create a revision if the last one was over 5 minutes ago
	var lastID int
	var lastCreatedAt time.Time
	err := h.DB.QueryRow("SELECT id, created_at FROM revisions WHERE document_id = ? ORDER BY created_at DESC LIMIT 1", id).Scan(&lastID, &lastCreatedAt)
	
	if err == sql.ErrNoRows || time.Since(lastCreatedAt) > 5*time.Minute {
		h.DB.Exec("INSERT INTO revisions (document_id, content, editor_id, change_summary) VALUES (?, ?, ?, ?)",
			id, content, 1, "Auto-snapshot")
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<span style="color: var(--primary);">Snapshot Synchronized</span>`))
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
				data := GetBaseData(r)
				data.Title = doc.Title
				data.ShowGate = true
				data.GateError = "Incorrect password. Please try again."
				data.Token = token

				tmpl := template.Must(template.ParseFiles("web/templates/index.html"))
				tmpl.Execute(w, data)
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
			data := GetBaseData(r)
			data.Title = doc.Title
			data.ShowGate = true
			data.Token = token

			tmpl := template.Must(template.ParseFiles("web/templates/index.html"))
			tmpl.Execute(w, data)
			return

		}
	}

	files, _ := GetDocumentFiles(h.DB, fmt.Sprintf("%d", doc.ID))

	data := GetBaseData(r)
	data.Title = doc.Title
	data.ShowGate = false
	data.Document = doc
	data.Files = files
	data.Token = token

	tmpl := template.Must(template.ParseFiles("web/templates/index.html"))
	tmpl.Execute(w, data)
}


