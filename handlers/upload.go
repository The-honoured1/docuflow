package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type UploadHandler struct {
	DB *sql.DB
}

// UploadFile handles multipart file uploads attached to a document or a folder.
func (h *UploadHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	docID := r.FormValue("document_id")
	folderID := r.FormValue("folder_id")

	if docID == "" && folderID == "" {
		// Root upload as a standalone file
	}

	// Limit upload size to 50 MB
	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		http.Error(w, "File too large (max 50 MB)", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Sanitize filename
	originalName := filepath.Base(header.Filename)
	ext := filepath.Ext(originalName)
	safeBase := sanitizeFilename(strings.TrimSuffix(originalName, ext))
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	safeName := fmt.Sprintf("%s_%s%s", safeBase, timestamp, ext)

	// Create upload directory
	// Use folder_id if docID is empty
	dirName := docID
	if dirName == "" {
		dirName = "standalone"
		if folderID != "" {
			dirName = "folder_" + folderID
		}
	}

	uploadDir := filepath.Join("uploads", dirName)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		http.Error(w, "Server error creating upload directory", http.StatusInternalServerError)
		return
	}

	destPath := filepath.Join(uploadDir, safeName)
	dst, err := os.Create(destPath)
	if err != nil {
		http.Error(w, "Server error saving file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(destPath)
		http.Error(w, "Server error writing file", http.StatusInternalServerError)
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	var res sql.Result
	if docID != "" {
		res, err = h.DB.Exec(
			`INSERT INTO document_files (document_id, file_name, file_path, mime_type, file_size) VALUES (?, ?, ?, ?, ?)`,
			docID, originalName, destPath, mimeType, written,
		)
	} else {
		if folderID == "" {
			res, err = h.DB.Exec(
				`INSERT INTO document_files (file_name, file_path, mime_type, file_size) VALUES (?, ?, ?, ?)`,
				originalName, destPath, mimeType, written,
			)
		} else {
			res, err = h.DB.Exec(
				`INSERT INTO document_files (folder_id, file_name, file_path, mime_type, file_size) VALUES (?, ?, ?, ?, ?)`,
				folderID, originalName, destPath, mimeType, written,
			)
		}
	}

	if err != nil {
		os.Remove(destPath)
		http.Error(w, "Failed to record file in database", http.StatusInternalServerError)
		return
	}

	// Redirect
	if docID != "" {
		http.Redirect(w, r, "/documents/view?id="+docID, http.StatusSeeOther)
	} else {
		dest := "/"
		if folderID != "" {
			dest = "/?folder_id=" + folderID
		}
		http.Redirect(w, r, dest, http.StatusSeeOther)
	}
}

// DeleteFile removes an uploaded file from disk and the database.
func (h *UploadHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fileID := r.FormValue("file_id")
	docID := r.FormValue("document_id")

	var filePath string
	err := h.DB.QueryRow("SELECT file_path FROM document_files WHERE id = ?", fileID).Scan(&filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		http.Error(w, "Failed to delete file from disk", http.StatusInternalServerError)
		return
	}

	h.DB.Exec("DELETE FROM document_files WHERE id = ?", fileID)

	http.Redirect(w, r, "/documents/view?id="+docID, http.StatusSeeOther)
}

// DownloadFile streams an uploaded file to the browser.
func (h *UploadHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	fileID := r.URL.Query().Get("file_id")

	var filePath, fileName, mimeType string
	err := h.DB.QueryRow("SELECT file_path, file_name, mime_type FROM document_files WHERE id = ?", fileID).
		Scan(&filePath, &fileName, &mimeType)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	http.ServeFile(w, r, filePath)
}

// sanitizeFilename strips dangerous characters from a filename base.
func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", "..", "_", " ", "_",
		"<", "_", ">", "_", ":", "_", "\"", "_",
		"|", "_", "?", "_", "*", "_",
	)
	result := replacer.Replace(name)
	if result == "" {
		return "file"
	}
	return result
}

// GetDocumentFiles returns all files attached to a document.
func GetDocumentFiles(db *sql.DB, docID string) ([]map[string]interface{}, error) {
	rows, err := db.Query(
		`SELECT id, file_name, mime_type, file_size, uploaded_at FROM document_files WHERE document_id = ? ORDER BY uploaded_at DESC`,
		docID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []map[string]interface{}
	for rows.Next() {
		var id int
		var fileName, mimeType string
		var fileSize int64
		var uploadedAt time.Time
		if err := rows.Scan(&id, &fileName, &mimeType, &fileSize, &uploadedAt); err != nil {
			continue
		}
		files = append(files, map[string]interface{}{
			"ID":         id,
			"FileName":   fileName,
			"MimeType":   mimeType,
			"FileSize":   HumanFileSize(fileSize),
			"UploadedAt": uploadedAt.Format("Jan 02, 2006"),
			"Icon":       FileIcon(mimeType),
		})
	}
	return files, nil
}

func HumanFileSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func FileIcon(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "🖼️"
	case mimeType == "application/pdf":
		return "📕"
	case strings.Contains(mimeType, "word") || strings.Contains(mimeType, "document"):
		return "📄"
	case strings.Contains(mimeType, "sheet") || strings.Contains(mimeType, "excel") || strings.Contains(mimeType, "csv"):
		return "📊"
	case strings.Contains(mimeType, "presentation") || strings.Contains(mimeType, "powerpoint"):
		return "📊"
	case strings.HasPrefix(mimeType, "text/"):
		return "📝"
	case strings.Contains(mimeType, "zip") || strings.Contains(mimeType, "archive") || strings.Contains(mimeType, "tar"):
		return "🗜️"
	default:
		return "📎"
	}
}
