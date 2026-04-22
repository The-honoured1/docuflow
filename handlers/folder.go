package handlers

import (
	"database/sql"
	"docuflow/models"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

type FolderHandler struct {
	DB *sql.DB
}

type DriveItem struct {
	ID          int
	Name        string
	Type        string // "folder", "document", "file"
	MimeType    string
	Size        string
	UpdatedAt   string
	Icon        string
	IsShared    bool
	IsProtected bool
}

func (h *FolderHandler) DriveView(w http.ResponseWriter, r *http.Request) {
	folderID := r.URL.Query().Get("folder_id")

	// Get Current Folder Name for Breadcrumbs
	currentFolderName := "My Drive"
	var parentID sql.NullInt64
	if folderID != "" {
		err := h.DB.QueryRow("SELECT name, parent_id FROM folders WHERE id = ?", folderID).Scan(&currentFolderName, &parentID)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}

	var items []DriveItem

	// 1. Get Folders
	folderQuery := "SELECT id, name, created_at FROM folders WHERE parent_id IS NULL"
	if folderID != "" {
		folderQuery = "SELECT id, name, created_at FROM folders WHERE parent_id = " + folderID
	}
	fRows, err := h.DB.Query(folderQuery)
	if err == nil {
		defer fRows.Close()
		for fRows.Next() {
			var f models.Folder
			fRows.Scan(&f.ID, &f.Name, &f.CreatedAt)
			items = append(items, DriveItem{
				ID:        f.ID,
				Name:      f.Name,
				Type:      "folder",
				UpdatedAt: f.CreatedAt.Format("Jan 02, 2006"),
				Icon:      "📁",
			})
		}
	}

	// 2. Get Documents
	docQuery := "SELECT id, title, updated_at, COALESCE(share_token,''), COALESCE(share_password,'') FROM documents WHERE folder_id IS NULL"
	if folderID != "" {
		docQuery = fmt.Sprintf("SELECT id, title, updated_at, COALESCE(share_token,''), COALESCE(share_password,'') FROM documents WHERE folder_id = %s", folderID)
	}
	dRows, err := h.DB.Query(docQuery)
	if err == nil {
		defer dRows.Close()
		for dRows.Next() {
			var d models.Document
			dRows.Scan(&d.ID, &d.Title, &d.UpdatedAt, &d.ShareToken, &d.SharePassword)
			items = append(items, DriveItem{
				ID:          d.ID,
				Name:        d.Title,
				Type:        "document",
				UpdatedAt:   d.UpdatedAt.Format("Jan 02, 2006"),
				Icon:        "📄",
				IsShared:    d.ShareToken != "",
				IsProtected: d.SharePassword != "",
			})
		}
	}

	// 3. Get Files
	fileQuery := "SELECT id, file_name, mime_type, file_size, uploaded_at FROM document_files WHERE folder_id IS NULL AND document_id IS NULL"
	if folderID != "" {
		fileQuery = fmt.Sprintf("SELECT id, file_name, mime_type, file_size, uploaded_at FROM document_files WHERE folder_id = %s AND document_id IS NULL", folderID)
	}
	fileRows, err := h.DB.Query(fileQuery)
	if err == nil {
		defer fileRows.Close()
		for fileRows.Next() {
			var id int
			var name, mime string
			var size int64
			var uploadedAt time.Time
			fileRows.Scan(&id, &name, &mime, &size, &uploadedAt)
			items = append(items, DriveItem{
				ID:        id,
				Name:      name,
				Type:      "file",
				MimeType:  mime,
				Size:      HumanFileSize(size),
				UpdatedAt: uploadedAt.Format("Jan 02, 2006"),
				Icon:      FileIcon(mime),
			})
		}
	}

	tmpl := template.Must(template.ParseFiles("web/templates/base.html", "web/templates/document_list.html"))
	tmpl.Execute(w, struct {
		User              string
		Items             []DriveItem
		CurrentFolderID   string
		CurrentFolderName string
		ParentFolderID    sql.NullInt64
	}{
		User:              GetBaseData(r).User,
		Items:             items,
		CurrentFolderID:   folderID,
		CurrentFolderName: currentFolderName,
		ParentFolderID:    parentID,
	})
}

func (h *FolderHandler) CreateFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.FormValue("name")
	parentID := r.FormValue("parent_id")
	ownerID := 1 // Hardcoded for now till auth is fully mapped

	if parentID == "" {
		_, err := h.DB.Exec("INSERT INTO folders (name, owner_id) VALUES (?, ?)", name, ownerID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		_, err := h.DB.Exec("INSERT INTO folders (name, parent_id, owner_id) VALUES (?, ?, ?)", name, parentID, ownerID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	dest := "/"
	if parentID != "" {
		dest = "/?folder_id=" + parentID
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}
