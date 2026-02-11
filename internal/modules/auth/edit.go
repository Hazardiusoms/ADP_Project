package auth

import (
	"database/sql"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
	"golang.org/x/crypto/bcrypt"
)

// EditUserHandler показывает страницу редактирования пользователя (админ)
func EditUserHandler(w http.ResponseWriter, r *http.Request) {
	user, err := GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if user.Role != "admin" {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}

	userIDStr := r.URL.Query().Get("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Redirect(w, r, "/admin/users?error="+url.QueryEscape("Неверный ID пользователя"), http.StatusSeeOther)
		return
	}

	// Get user data
	var editUser struct {
		ID        int
		Email     string
		FullName  string
		Phone     string
		Role      string
		IsActive  bool
		CreatedAt string
	}

	err = database.SafeQueryRow(
		"SELECT id, email, full_name, phone, role, is_active, created_at FROM users WHERE id = ?",
		userID,
	).Scan(&editUser.ID, &editUser.Email, &editUser.FullName, &editUser.Phone, &editUser.Role, &editUser.IsActive, &editUser.CreatedAt)

	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/admin/users?error="+url.QueryEscape("Пользователь не найден"), http.StatusSeeOther)
		return
	}

	errorMsg := r.URL.Query().Get("error")
	successMsg := r.URL.Query().Get("success")

	data := map[string]interface{}{
		"User":    user,
		"EditUser": editUser,
		"Error":   errorMsg,
		"Success": successMsg,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/edit_user.html"))
	tmpl.ExecuteTemplate(w, "layout.html", data)
}

// UpdateUserHandler обновляет пользователя (админ)
func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if user.Role != "admin" {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}

	userIDStr := r.FormValue("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Redirect(w, r, "/admin/users/edit?id="+userIDStr+"&error="+url.QueryEscape("Неверный ID"), http.StatusSeeOther)
		return
	}

	fullName := r.FormValue("full_name")
	phone := r.FormValue("phone")
	role := r.FormValue("role")
	isActive := r.FormValue("is_active") == "1"
	newPassword := r.FormValue("new_password")

	// Build update query
	updates := []string{}
	args := []interface{}{}

	if fullName != "" {
		updates = append(updates, "full_name = ?")
		args = append(args, fullName)
	}

	if phone != "" {
		updates = append(updates, "phone = ?")
		args = append(args, phone)
	}

	if role != "" {
		updates = append(updates, "role = ?")
		args = append(args, role)
	}

	updates = append(updates, "is_active = ?")
	args = append(args, isActive)

	if newPassword != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err == nil {
			updates = append(updates, "password_hash = ?")
			args = append(args, string(hashedPassword))
		}
	}

	if len(updates) == 0 {
		http.Redirect(w, r, "/admin/users/edit?id="+userIDStr+"&error="+url.QueryEscape("Нет изменений"), http.StatusSeeOther)
		return
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, userID)

	query := "UPDATE users SET " + strings.Join(updates, ", ") + " WHERE id = ?"
	_, err = database.SafeExec(query, args...)
	if err != nil {
		http.Redirect(w, r, "/admin/users/edit?id="+userIDStr+"&error="+url.QueryEscape("Ошибка обновления"), http.StatusSeeOther)
		return
	}

	// Log audit
	database.SafeExec(
		`INSERT INTO audit_logs (action, user_id, entity_type, entity_id, new_values, ip_address) 
		 VALUES (?, ?, 'user', ?, ?, ?)`,
		"UPDATE", user.ID, userID, `{"role":"`+role+`","is_active":`+strconv.FormatBool(isActive)+`}`, r.RemoteAddr,
	)

	http.Redirect(w, r, "/admin/users/edit?id="+userIDStr+"&success="+url.QueryEscape("Пользователь успешно обновлен"), http.StatusSeeOther)
}
