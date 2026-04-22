package handlers

import (
	"database/sql"
	"docuflow/models"
	"html/template"
	"net/http"
)

type SearchHandler struct {
	DB *sql.DB
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	if query == "" {
		tmpl := template.Must(template.ParseFiles("web/templates/index.html"))
		data := GetBaseData(r)
		data.Query = ""
		data.View = "search"
		tmpl.Execute(w, data)
		return
	}

	// Search documents by title and content
	rows, err := h.DB.Query(`
		SELECT id, title, content, updated_at 
		FROM documents 
		WHERE title LIKE ? OR content LIKE ?
		ORDER BY updated_at DESC
		LIMIT 50`, "%"+query+"%", "%"+query+"%")
	if err != nil {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []models.Document
	for rows.Next() {
		var d models.Document
		if err := rows.Scan(&d.ID, &d.Title, &d.Content, &d.UpdatedAt); err != nil {
			continue
		}
		// Truncate content for preview
		if len(d.Content) > 200 {
			d.Content = d.Content[:200] + "..."
		}
		results = append(results, d)
	}

	data := GetBaseData(r)
	data.Query = query
	data.Results = results
	data.View = "search"

	tmpl := template.Must(template.ParseFiles("web/templates/index.html"))
	tmpl.Execute(w, data)
}

