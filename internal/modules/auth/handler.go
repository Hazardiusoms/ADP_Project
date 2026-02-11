package auth

import (
	"database/sql"
	"html/template"
	"net/http"
	"time"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
	"golang.org/x/crypto/bcrypt"
)

// Matches User Class
type User struct {
	ID           int
	Email        string
	PasswordHash string
	FullName     string
	Phone        string
	Role         string
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/register.html"))
		tmpl.ExecuteTemplate(w, "layout.html", nil)
		return
	}

	// Capture fields from ERD
	email := r.FormValue("email")
	password := r.FormValue("password")
	fullName := r.FormValue("full_name")
	phone := r.FormValue("phone")

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	_, err := database.DB.Exec("INSERT INTO users (email, password_hash, full_name, phone, role) VALUES (?, ?, ?, ?, ?)",
		email, string(hashedPassword), fullName, phone, "user")

	if err != nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/login.html"))
		tmpl.ExecuteTemplate(w, "layout.html", nil)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	var dbPassword string
	var userID int
	// Query matches Login() method in Class Diagram
	err := database.DB.QueryRow("SELECT id, password_hash FROM users WHERE email = ?", email).Scan(&userID, &dbPassword)

	if err == sql.ErrNoRows || bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password)) != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Simple Session Cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "user_session",
		Value:   email,
		Expires: time.Now().Add(24 * time.Hour),
		Path:    "/",
	})

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
