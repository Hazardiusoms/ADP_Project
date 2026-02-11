package auth

import (
	"database/sql"
	"html/template"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
	"golang.org/x/crypto/bcrypt"
)

// AdminUsersHandler shows list of all users (for debugging/admin)
func AdminUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Simple admin check - in production, use proper admin middleware
	user, err := GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login?error="+url.QueryEscape("Необходима авторизация"), http.StatusSeeOther)
		return
	}

	if user.Role != "admin" {
		http.Error(w, "Доступ запрещен. Требуются права администратора.", http.StatusForbidden)
		return
	}

	// Get all users
	rows, err := database.SafeQuery("SELECT id, email, full_name, phone, role, created_at, is_active FROM users ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, "Ошибка получения пользователей", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type UserInfo struct {
		ID        int
		Email     string
		FullName  string
		Phone     string
		Role      string
		CreatedAt string
		IsActive  bool
	}

	var users []UserInfo
	for rows.Next() {
		var u UserInfo
		var createdAt string
		err := rows.Scan(&u.ID, &u.Email, &u.FullName, &u.Phone, &u.Role, &createdAt, &u.IsActive)
		if err != nil {
			continue
		}
		u.CreatedAt = createdAt
		users = append(users, u)
	}

	// Get error/success messages from URL
	errorMsg := r.URL.Query().Get("error")
	successMsg := r.URL.Query().Get("success")

	data := map[string]interface{}{
		"User":    user,
		"Users":   users,
		"Error":   errorMsg,
		"Success": successMsg,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/admin_users.html"))
	tmpl.ExecuteTemplate(w, "layout.html", data)
}

// DeleteUserHandler deletes a user (for admin/debugging)
func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login?error="+url.QueryEscape("Необходима авторизация"), http.StatusSeeOther)
		return
	}

	if user.Role != "admin" {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}

	userIDStr := r.FormValue("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Redirect(w, r, "/admin/users?error="+url.QueryEscape("Неверный ID пользователя"), http.StatusSeeOther)
		return
	}

	// First, delete all bookings by this user
	_, err = database.SafeExec("DELETE FROM bookings WHERE user_id = ?", userID)
	if err != nil {
		// Log but continue - might not have bookings
	}

	// Delete or transfer resources created by this user
	// We'll delete them for now
	_, err = database.SafeExec("DELETE FROM resources WHERE created_by = ?", userID)
	if err != nil {
		// Log but continue
	}

	// Delete payments related to user's bookings (orphaned payments)
	_, err = database.SafeExec(`
		DELETE FROM payments 
		WHERE booking_id NOT IN (SELECT id FROM bookings)
	`)
	if err != nil {
		// Log but continue
	}

	// Delete notifications for this user
	_, err = database.SafeExec("DELETE FROM notifications WHERE user_id = ?", userID)
	if err != nil {
		// Log but continue
	}

	// Finally, delete the user
	_, err = database.SafeExec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		http.Redirect(w, r, "/admin/users?error="+url.QueryEscape("Ошибка удаления пользователя: "+err.Error()), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/users?success="+url.QueryEscape("Пользователь успешно удален"), http.StatusSeeOther)
}

// ResetPasswordHandler allows resetting password (for testing)
func ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	email := r.FormValue("email")
	newPassword := r.FormValue("new_password")

	if email == "" || newPassword == "" {
		http.Redirect(w, r, "/admin/users?error="+url.QueryEscape("Email и новый пароль обязательны"), http.StatusSeeOther)
		return
	}

	// Check if user exists
	var userID int
	err := database.SafeQueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&userID)
	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/admin/users?error="+url.QueryEscape("Пользователь с таким email не найден"), http.StatusSeeOther)
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Redirect(w, r, "/admin/users?error="+url.QueryEscape("Ошибка обработки пароля"), http.StatusSeeOther)
		return
	}

	// Update password
	_, err = database.SafeExec("UPDATE users SET password_hash = ? WHERE email = ?", string(hashedPassword), email)
	if err != nil {
		http.Redirect(w, r, "/admin/users?error="+url.QueryEscape("Ошибка обновления пароля"), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/users?success="+url.QueryEscape("Пароль успешно изменен для "+email), http.StatusSeeOther)
}
