package main

import (
	"database/sql"
	"docuflow/db"
	"docuflow/handlers"
	"log"
	"net/http"
	"os"
)

type Config struct {
	Port string
	DB   *sql.DB
}

func main() {
	// Initialize Database
	database, err := db.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Ensure uploads directory exists
	if err := os.MkdirAll("uploads", 0755); err != nil {
		log.Fatalf("Failed to create uploads directory: %v", err)
	}

	// Initialize Router
	mux := http.NewServeMux()

	// Static Files
	fs := http.FileServer(http.Dir("./web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Handlers
	authHandler := &handlers.AuthHandler{DB: database}
	docHandler := &handlers.DocumentHandler{DB: database}
	revHandler := &handlers.RevisionHandler{DB: database}
	commentHandler := &handlers.CommentHandler{DB: database}
	searchHandler := &handlers.SearchHandler{DB: database}
	uploadHandler := &handlers.UploadHandler{DB: database}
	folderHandler := &handlers.FolderHandler{DB: database}

	// Drive / Folder routes
	mux.HandleFunc("/", folderHandler.DriveView)
	mux.HandleFunc("/folders/new", folderHandler.CreateFolder)

	// Document routes
	mux.HandleFunc("/documents/new", docHandler.NewDocument)
	mux.HandleFunc("/documents/view", docHandler.ViewDocument)
	mux.HandleFunc("/documents/edit", docHandler.EditDocument)
	mux.HandleFunc("/documents/autosave", docHandler.Autosave)
	mux.HandleFunc("/documents/set-password", docHandler.SetPassword)
	mux.HandleFunc("/documents/share", docHandler.GenerateShareLink)

	// Share link (public, no auth required)
	mux.HandleFunc("/share/", docHandler.ShareView)

	// File upload routes
	mux.HandleFunc("/documents/upload", uploadHandler.UploadFile)
	mux.HandleFunc("/documents/delete-file", uploadHandler.DeleteFile)
	mux.HandleFunc("/files/download", uploadHandler.DownloadFile)

	// Revision routes
	mux.HandleFunc("/revisions", revHandler.ListRevisions)
	mux.HandleFunc("/revisions/view", revHandler.ViewRevision)
	mux.HandleFunc("/revisions/rollback", revHandler.Rollback)

	// Comment routes
	mux.HandleFunc("/comments", commentHandler.ListComments)
	mux.HandleFunc("/comments/add", commentHandler.AddComment)
	mux.HandleFunc("/comments/delete", commentHandler.DeleteComment)

	// Search
	mux.HandleFunc("/search", searchHandler.Search)

	// Auth
	mux.HandleFunc("/register", authHandler.Register)
	mux.HandleFunc("/login", authHandler.Login)
	mux.HandleFunc("/logout", authHandler.Logout)

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
