package handlers

import (
	"database/sql"
	"docuflow/models"
	"html/template"
	"net/http"
)

// PageData holds all potential information needed by index.html to prevent template execution errors.
type PageData struct {
	User              string
	View              string
	Query             string
	CurrentFolderID   string
	CurrentFolderName string
	ParentFolderID    sql.NullInt64
	Items             any
	Documents         any
	Document          models.Document
	Content           template.HTML
	Files             any
	ShareURL          string
	IsProtected       bool
	FolderID          string
	Revisions         any
	Results           any
	Title             string
	ShowGate          bool
	GateError         string
	Token             string
	Comments          any
	DocumentID        any
}

// GetBaseData extracts common fields and returns an initialized PageData object.
func GetBaseData(r *http.Request) PageData {
	user := ""
	cookie, err := r.Cookie("session_token")
	if err == nil {
		user = cookie.Value
	}

	return PageData{
		User: user,
	}
}

