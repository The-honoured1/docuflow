package models

import (
	"time"
)

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // Store hashed password
	Role      string    `json:"role"` // "admin", "editor", "viewer"
	CreatedAt time.Time `json:"created_at"`
}

type Document struct {
	ID            int       `json:"id"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	OwnerID       int       `json:"owner_id"`
	FolderID      *int      `json:"folder_id"` // NULL if root
	ShareToken    string    `json:"share_token"`
	SharePassword string    `json:"-"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Revision struct {
	ID            int       `json:"id"`
	DocumentID    int       `json:"document_id"`
	Content       string    `json:"content"`
	EditorID      int       `json:"editor_id"`
	CreatedAt     time.Time `json:"created_at"`
	ChangeSummary string    `json:"change_summary"`
}

type Folder struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	ParentID  *int      `json:"parent_id"` // NULL if root
	OwnerID   int       `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

type DocumentFile struct {
	ID         int       `json:"id"`
	DocumentID *int      `json:"document_id"` // Now optional
	FolderID   *int      `json:"folder_id"`   // New: for standalone files
	FileName   string    `json:"file_name"`
	FilePath   string    `json:"file_path"`
	MimeType   string    `json:"mime_type"`
	FileSize   int64     `json:"file_size"`
	UploadedAt time.Time `json:"uploaded_at"`
}
