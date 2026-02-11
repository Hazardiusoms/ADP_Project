package auth

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
	"golang.org/x/crypto/bcrypt"
)

// ProfileHandler показывает личный кабинет пользователя
func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	user, err := GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get user details with payment info
	var paymentDetails sql.NullString
	err = database.SafeQueryRow(
		"SELECT payment_details FROM users WHERE id = ?",
		user.ID,
	).Scan(&paymentDetails)

	var paymentData map[string]interface{}
	if paymentDetails.Valid && paymentDetails.String != "" {
		json.Unmarshal([]byte(paymentDetails.String), &paymentData)
	}

	errorMsg := r.URL.Query().Get("error")
	successMsg := r.URL.Query().Get("success")

	data := map[string]interface{}{
		"User":          user,
		"PaymentDetails": paymentData,
		"Error":         errorMsg,
		"Success":       successMsg,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/profile.html"))
	tmpl.ExecuteTemplate(w, "layout.html", data)
}

// UpdateProfileHandler обновляет профиль пользователя
func UpdateProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	fullName := r.FormValue("full_name")
	phone := r.FormValue("phone")
	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")

	// Payment details
	cardNumber := r.FormValue("card_number")
	cardHolder := r.FormValue("card_holder")
	expiryDate := r.FormValue("expiry_date")
	cvv := r.FormValue("cvv")
	paymentMethod := r.FormValue("payment_method")

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

	// Update password if provided
	if newPassword != "" {
		// Verify current password
		var passwordHash string
		err = database.SafeQueryRow("SELECT password_hash FROM users WHERE id = ?", user.ID).Scan(&passwordHash)
		if err == nil {
			err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(currentPassword))
			if err != nil {
				http.Redirect(w, r, "/profile?error="+url.QueryEscape("Неверный текущий пароль"), http.StatusSeeOther)
				return
			}

			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
			if err == nil {
				updates = append(updates, "password_hash = ?")
				args = append(args, string(hashedPassword))
			}
		}
	}

	// Update payment details
	if cardNumber != "" || paymentMethod != "" {
		paymentData := map[string]interface{}{
			"card_number":   cardNumber,
			"card_holder":   cardHolder,
			"expiry_date":   expiryDate,
			"cvv":           cvv,
			"payment_method": paymentMethod,
		}
		paymentJSON, _ := json.Marshal(paymentData)
		updates = append(updates, "payment_details = ?")
		args = append(args, string(paymentJSON))
	}

	if len(updates) == 0 {
		http.Redirect(w, r, "/profile?error="+url.QueryEscape("Нет изменений для сохранения"), http.StatusSeeOther)
		return
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, user.ID)

	query := "UPDATE users SET " + strings.Join(updates, ", ") + " WHERE id = ?"
	_, err = database.SafeExec(query, args...)
	if err != nil {
		http.Redirect(w, r, "/profile?error="+url.QueryEscape("Ошибка обновления профиля"), http.StatusSeeOther)
		return
	}

	// Log audit
	database.SafeExec(
		`INSERT INTO audit_logs (action, user_id, entity_type, entity_id, new_values, ip_address) 
		 VALUES (?, ?, 'user', ?, ?, ?)`,
		"UPDATE_PROFILE", user.ID, user.ID, `{"updated":"profile"}`, r.RemoteAddr,
	)

	http.Redirect(w, r, "/profile?success="+url.QueryEscape("Профиль успешно обновлен"), http.StatusSeeOther)
}
