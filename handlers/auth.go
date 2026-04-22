package handlers

import (
	"database/sql"
	"docuflow/models"
	"html/template"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	DB *sql.DB
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl := template.Must(template.ParseFiles("web/templates/index.html"))
		tmpl.Execute(w, struct {
			User string
			View string
		}{
			User: GetBaseData(r).User,
			View: "register",
		})
		return
	}

	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	_, err = h.DB.Exec("INSERT INTO users (username, email, password, role, created_at) VALUES (?, ?, ?, ?, ?)",
		username, email, string(hashedPassword), "editor", time.Now())

	if err != nil {
		http.Error(w, "Username or email already exists", http.StatusConflict)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl := template.Must(template.ParseFiles("web/templates/index.html"))
		tmpl.Execute(w, struct {
			User string
			View string
		}{
			User: GetBaseData(r).User,
			View: "login",
		})
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	var user models.User
	err := h.DB.QueryRow("SELECT id, password, role FROM users WHERE username = ?", username).Scan(&user.ID, &user.Password, &user.Role)
	if err == sql.ErrNoRows {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Set session cookie (Basic implementation for now)
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    username, // In real app, use a secure token
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
