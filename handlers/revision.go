package handlers

import (
	"database/sql"
	"docuflow/models"
	"html/template"
	"net/http"
	"strconv"
	"time"
)

type RevisionHandler struct {
	DB *sql.DB
}

func (h *RevisionHandler) ListRevisions(w http.ResponseWriter, r *http.Request) {
	docID := r.URL.Query().Get("doc_id")

	rows, err := h.DB.Query(`
		SELECT id, document_id, created_at, change_summary 
		FROM revisions 
		WHERE document_id = ? 
		ORDER BY created_at DESC`, docID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var revisions []models.Revision
	for rows.Next() {
		var rev models.Revision
		if err := rows.Scan(&rev.ID, &rev.DocumentID, &rev.CreatedAt, &rev.ChangeSummary); err != nil {
			continue
		}
		revisions = append(revisions, rev)
	}

	// Get document title
	var title string
	h.DB.QueryRow("SELECT title FROM documents WHERE id = ?", docID).Scan(&title)

	tmpl := template.Must(template.ParseFiles("web/templates/index.html"))
	data := struct {
		User       string
		DocumentID string
		Title      string
		Revisions  []models.Revision
		View       string
	}{
		User:       GetBaseData(r).User,
		DocumentID: docID,
		Title:      title,
		Revisions:  revisions,
		View:       "revisions",
	}
	tmpl.Execute(w, data)
}

func (h *RevisionHandler) ViewRevision(w http.ResponseWriter, r *http.Request) {
	revID := r.URL.Query().Get("id")

	var rev models.Revision
	err := h.DB.QueryRow(`
		SELECT id, document_id, content, created_at, change_summary 
		FROM revisions 
		WHERE id = ?`, revID).Scan(&rev.ID, &rev.DocumentID, &rev.Content, &rev.CreatedAt, &rev.ChangeSummary)
	if err != nil {
		http.Error(w, "Revision not found", http.StatusNotFound)
		return
	}

	// Get document title
	var title string
	h.DB.QueryRow("SELECT title FROM documents WHERE id = ?", rev.DocumentID).Scan(&title)

	tmpl := template.Must(template.ParseFiles("web/templates/index.html"))
	data := struct {
		User     string
		Title    string
		Revision models.Revision
		View     string
	}{
		User:     GetBaseData(r).User,
		Title:    title,
		Revision: rev,
		View:     "revision_view",
	}
	tmpl.Execute(w, data)
}

func (h *RevisionHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	revID := r.FormValue("revision_id")

	// Get revision content
	var content string
	var docID int
	err := h.DB.QueryRow("SELECT content, document_id FROM revisions WHERE id = ?", revID).Scan(&content, &docID)
	if err != nil {
		http.Error(w, "Revision not found", http.StatusNotFound)
		return
	}

	// Update document with revision content
	_, err = h.DB.Exec("UPDATE documents SET content = ?, updated_at = ? WHERE id = ?", content, time.Now(), docID)
	if err != nil {
		http.Error(w, "Failed to rollback", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/documents/view?id="+strconv.Itoa(docID), http.StatusSeeOther)
}
