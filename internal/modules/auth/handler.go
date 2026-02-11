package auth

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
	"github.com/Hazardiusoms/ADP_Project/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// RegisterHandler обрабатывает регистрацию пользователя (html форма и json api)
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	// если get запрос - показываем форму
	if r.Method == "GET" {
		// получаем сообщения об ошибках/успехе из url
		errorMsg := r.URL.Query().Get("error")
		successMsg := r.URL.Query().Get("success")

		data := map[string]interface{}{
			"Error":   errorMsg,
			"Success": successMsg,
		}

		// рендерим шаблон регистрации
		tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/register.html"))
		tmpl.ExecuteTemplate(w, "layout.html", data)
		return
	}

	// проверяем если это json запрос
	if r.Header.Get("Content-Type") == "application/json" {
		handleJSONRegister(w, r)
		return
	}

	// парсим форму
	if err := SafeParseForm(r); err != nil {
		log.Printf("Error parsing form in RegisterHandler: %v", err)
		http.Redirect(w, r, "/register?error="+url.QueryEscape("Error processing form: "+err.Error()), http.StatusSeeOther)
		return
	}

	// получаем данные из формы
	email := r.FormValue("email")
	password := r.FormValue("password")
	fullName := r.FormValue("full_name")
	phone := r.FormValue("phone")

	log.Printf("Register attempt - Email: %s, FullName: %s", email, fullName)

	// проверяем обязательные поля
	if email == "" || password == "" || fullName == "" {
		http.Redirect(w, r, "/register?error="+url.QueryEscape("Please fill in all required fields"), http.StatusSeeOther)
		return
	}

	// проверяем, существует ли пользователь с таким email
	var existingEmail string
	err := database.SafeQueryRow(
		"SELECT email FROM users WHERE email = ?",
		email,
	).Scan(&existingEmail)

	// если пользователь найден - ошибка
	if err == nil {
		// пользователь уже существует
		http.Redirect(w, r, "/register?error="+url.QueryEscape("User with email '"+email+"' is already registered. Please use a different email or login."), http.StatusSeeOther)
		return
	}

	// если ошибка не "no rows", значит проблема с бд
	if err != sql.ErrNoRows {
		log.Printf("Database error during registration check: %v", err)
		http.Redirect(w, r, "/register?error="+url.QueryEscape("Database error. Please try again."), http.StatusSeeOther)
		return
	}

	// хешируем пароль через bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Redirect(w, r, "/register?error="+url.QueryEscape("Error processing password. Please try again."), http.StatusSeeOther)
		return
	}

	// создаем пользователя в бд
	_, err = database.SafeExec(
		"INSERT INTO users (email, password_hash, full_name, phone, role) VALUES (?, ?, ?, ?, ?)",
		email, string(hashedPassword), fullName, phone, "user",
	)
	if err != nil {
		log.Printf("Error inserting user: %v", err)
		http.Redirect(w, r, "/register?error="+url.QueryEscape("Error creating user. Please try again or use a different email."), http.StatusSeeOther)
		return
	}

	// редирект на логин с сообщением об успехе
	http.Redirect(w, r, "/login?success="+url.QueryEscape("Registration successful! You can now login."), http.StatusSeeOther)
}

// handleJSONRegister обрабатывает json запросы на регистрацию
func handleJSONRegister(w http.ResponseWriter, r *http.Request) {
	// парсим json из тела запроса
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	// проверяем обязательные поля
	if req.Email == "" || req.Password == "" || req.FullName == "" {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Missing required fields",
		})
		return
	}

	// хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Error processing password",
		})
		return
	}

	// создаем пользователя
	result, err := database.SafeExec(
		"INSERT INTO users (email, password_hash, full_name, phone, role) VALUES (?, ?, ?, ?, ?)",
		req.Email, string(hashedPassword), req.FullName, req.Phone, "user",
	)
	if err != nil {
		// если ошибка - скорее всего пользователь уже существует
		respondJSON(w, http.StatusConflict, models.APIResponse{
			Success: false,
			Error:   "User already exists",
		})
		return
	}

	// получаем id созданного пользователя
	userID, _ := result.LastInsertId()
	respondJSON(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "User registered successfully",
		Data: map[string]interface{}{
			"user_id": userID,
			"email":   req.Email,
		},
	})
}

// LoginHandler обрабатывает вход пользователя (html форма и json api)
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	// если get запрос - показываем форму входа
	if r.Method == "GET" {
		// получаем сообщения из url
		errorMsg := r.URL.Query().Get("error")
		successMsg := r.URL.Query().Get("success")

		data := map[string]interface{}{
			"Error":   errorMsg,
			"Success": successMsg,
		}

		// рендерим шаблон логина
		tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/login.html"))
		tmpl.ExecuteTemplate(w, "layout.html", data)
		return
	}

	// проверяем если это json запрос
	if r.Header.Get("Content-Type") == "application/json" {
		handleJSONLogin(w, r)
		return
	}

	// обрабатываем отправку формы
	email := r.FormValue("email")
	password := r.FormValue("password")

	// проверяем обязательные поля
	if email == "" || password == "" {
		http.Redirect(w, r, "/login?error="+url.QueryEscape("Please enter email and password"), http.StatusSeeOther)
		return
	}

	// ищем пользователя в бд
	var dbPassword string
	var userID int
	var fullName string
	err := database.SafeQueryRow(
		"SELECT id, password_hash, full_name FROM users WHERE email = ? AND is_active = 1",
		email,
	).Scan(&userID, &dbPassword, &fullName)

	// проверяем пароль или если пользователь не найден
	if err == sql.ErrNoRows || bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password)) != nil {
		http.Redirect(w, r, "/login?error="+url.QueryEscape("Invalid email or password. Please check your credentials and try again."), http.StatusSeeOther)
		return
	}

	// устанавливаем куку с сессией
	http.SetCookie(w, &http.Cookie{
		Name:    "user_session",
		Value:   email,
		Expires: time.Now().Add(24 * time.Hour),
		Path:    "/",
	})
	// редирект на дашборд
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// handleJSONLogin handles JSON login requests
func handleJSONLogin(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	var dbPassword string
	var userID int
	var fullName, role string
	err := database.SafeQueryRow(
		"SELECT id, password_hash, full_name, role FROM users WHERE email = ? AND is_active = 1",
		req.Email,
	).Scan(&userID, &dbPassword, &fullName, &role)

	if err == sql.ErrNoRows || bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(req.Password)) != nil {
		respondJSON(w, http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

		// устанавливаем куку с сессией
		http.SetCookie(w, &http.Cookie{
		Name:    "user_session",
		Value:   req.Email,
		Expires: time.Now().Add(24 * time.Hour),
		Path:    "/",
	})

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Login successful",
		Data: map[string]interface{}{
			"user_id":   userID,
			"email":     req.Email,
			"full_name": fullName,
			"role":      role,
		},
	})
}

// GetCurrentUser возвращает текущего залогиненного пользователя
func GetCurrentUser(r *http.Request) (*models.User, error) {
	// получаем куку с сессией
	cookie, err := r.Cookie("user_session")
	if err != nil {
		return nil, err
	}

	// ищем пользователя по email из куки
	var user models.User
	err = database.SafeQueryRow(
		"SELECT id, email, full_name, phone, role, created_at, updated_at, is_active FROM users WHERE email = ?",
		cookie.Value,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.Phone, &user.Role, &user.CreatedAt, &user.UpdatedAt, &user.IsActive)

	if err != nil {
		return nil, err
	}
	return &user, nil
}

// respondJSON отправляет json ответ
func respondJSON(w http.ResponseWriter, statusCode int, response models.APIResponse) {
	// устанавливаем заголовок и статус
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	// кодируем и отправляем json
	json.NewEncoder(w).Encode(response)
}
