package db

import (
	"database/sql"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

func InitDB() (*sql.DB, error) {
	dbPath := "./docuflow.db"

	// Ensure file exists or can be created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		file, err := os.Create(dbPath)
		if err != nil {
			return nil, err
		}
		file.Close()
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := InitSchema(db); err != nil {
		return nil, err
	}

	log.Println("Connected to SQLite database at", dbPath)
	return db, nil
}

func InitSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		role TEXT DEFAULT 'editor',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS folders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		parent_id INTEGER,
		owner_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(parent_id) REFERENCES folders(id),
		FOREIGN KEY(owner_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS documents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		content TEXT,
		owner_id INTEGER,
		folder_id INTEGER,
		share_token TEXT UNIQUE,
		share_password TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(owner_id) REFERENCES users(id),
		FOREIGN KEY(folder_id) REFERENCES folders(id)
	);

	CREATE TABLE IF NOT EXISTS revisions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		document_id INTEGER,
		content TEXT,
		editor_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		change_summary TEXT,
		FOREIGN KEY(document_id) REFERENCES documents(id),
		FOREIGN KEY(editor_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		document_id INTEGER,
		user_id INTEGER,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(document_id) REFERENCES documents(id),
		FOREIGN KEY(user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS document_files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		document_id INTEGER, -- Removed NOT NULL to allow standalone files
		folder_id INTEGER,
		file_name TEXT NOT NULL,
		file_path TEXT NOT NULL,
		mime_type TEXT,
		file_size INTEGER,
		uploaded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(document_id) REFERENCES documents(id),
		FOREIGN KEY(folder_id) REFERENCES folders(id)
	);
	`
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	// Safely migrate existing databases — ignore errors if columns already exist
	db.Exec(`ALTER TABLE documents ADD COLUMN share_token TEXT`)
	db.Exec(`ALTER TABLE documents ADD COLUMN share_password TEXT`)
	db.Exec(`ALTER TABLE documents ADD COLUMN folder_id INTEGER`)
	db.Exec(`ALTER TABLE document_files ADD COLUMN folder_id INTEGER`)
	return nil
}
