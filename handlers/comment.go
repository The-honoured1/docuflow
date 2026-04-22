package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"time"
)

type Comment struct {
	ID         int
	DocumentID int
	UserID     int
	Username   string
	Content    string
	CreatedAt  time.Time
}

type CommentHandler struct {
	DB *sql.DB
}

func (h *CommentHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	docID := r.URL.Query().Get("doc_id")

	rows, err := h.DB.Query(`
		SELECT c.id, c.content, c.created_at, u.username 
		FROM comments c
		LEFT JOIN users u ON c.user_id = u.id
		WHERE c.document_id = ?
		ORDER BY c.created_at DESC`, docID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		var username sql.NullString
		if err := rows.Scan(&c.ID, &c.Content, &c.CreatedAt, &username); err != nil {
			continue
		}
		if username.Valid {
			c.Username = username.String
		} else {
			c.Username = "Anonymous"
		}
		comments = append(comments, c)
	}

	// Return HTML partial from the unified template
	tmpl := template.Must(template.ParseFiles("web/templates/index.html"))
	tmpl.ExecuteTemplate(w, "comments", struct {
		DocumentID string
		Comments   []Comment
	}{
		DocumentID: docID,
		Comments:   comments,
	})
}

func (h *CommentHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	docID := r.FormValue("document_id")
	content := r.FormValue("content")
	userID := 1 // Hardcoded for now

	_, err := h.DB.Exec("INSERT INTO comments (document_id, user_id, content) VALUES (?, ?, ?)",
		docID, userID, content)
	if err != nil {
		http.Error(w, "Failed to add comment", http.StatusInternalServerError)
		return
	}

	// Return updated comments list
	h.ListComments(w, r)
}

func (h *CommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	commentID := r.FormValue("comment_id")
	docID := r.FormValue("document_id")

	_, err := h.DB.Exec("DELETE FROM comments WHERE id = ?", commentID)
	if err != nil {
		http.Error(w, "Failed to delete comment", http.StatusInternalServerError)
		return
	}

	// Redirect back to refresh comments
	r.URL.RawQuery = "doc_id=" + docID
	h.ListComments(w, r)
}
