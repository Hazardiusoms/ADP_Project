package auth

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
)

// AuditLogInfo содержит информацию об аудит-логе для отображения
type AuditLogInfo struct {
	ID         int
	Action     string
	UserName   string
	UserEmail  string
	EntityType string
	EntityID   int
	OldValues  string
	NewValues  string
	CreatedAt  string
	IPAddress  string
}

// AdminAuditLogHandler показывает журнал аудита
func AdminAuditLogHandler(w http.ResponseWriter, r *http.Request) {
	user, err := GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login?error="+url.QueryEscape("Необходима авторизация"), http.StatusSeeOther)
		return
	}

	if user.Role != "admin" {
		http.Error(w, "Доступ запрещен. Требуются права администратора.", http.StatusForbidden)
		return
	}

	// Получаем все аудит-логи с информацией о пользователях
	rows, err := database.SafeQuery(
		`SELECT 
			a.id, 
			a.action, 
			COALESCE(u.full_name, 'System') as user_name,
			COALESCE(u.email, '') as user_email,
			a.entity_type, 
			a.entity_id, 
			a.old_values, 
			a.new_values, 
			a.created_at, 
			a.ip_address
		FROM audit_logs a
		LEFT JOIN users u ON a.user_id = u.id
		ORDER BY a.created_at DESC
		LIMIT 500`,
	)
	if err != nil {
		http.Error(w, "Ошибка получения журнала аудита", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []AuditLogInfo
	for rows.Next() {
		var log AuditLogInfo
		var oldValues, newValues sql.NullString
		err := rows.Scan(
			&log.ID,
			&log.Action,
			&log.UserName,
			&log.UserEmail,
			&log.EntityType,
			&log.EntityID,
			&oldValues,
			&newValues,
			&log.CreatedAt,
			&log.IPAddress,
		)
		if err != nil {
			continue
		}
		if oldValues.Valid {
			log.OldValues = oldValues.String
		}
		if newValues.Valid {
			log.NewValues = newValues.String
		}
		logs = append(logs, log)
	}

	// Get error/success message from URL
	errorMsg := r.URL.Query().Get("error")
	successMsg := r.URL.Query().Get("success")

	data := map[string]interface{}{
		"User":    user,
		"Logs":    logs,
		"Error":   errorMsg,
		"Success": successMsg,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/admin_audit.html"))
	tmpl.ExecuteTemplate(w, "layout.html", data)
}

// DeleteResourceAdminHandler удаляет ресурс (только админ)
func DeleteResourceAdminHandler(w http.ResponseWriter, r *http.Request) {
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

	resourceIDStr := r.FormValue("resource_id")
	if resourceIDStr == "" {
		http.Redirect(w, r, "/admin/audit?error="+url.QueryEscape("ID ресурса не указан"), http.StatusSeeOther)
		return
	}

	// Soft delete resource
	_, err = database.SafeExec(
		"UPDATE resources SET is_active = 0, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		resourceIDStr,
	)
	if err != nil {
		http.Redirect(w, r, "/admin/audit?error="+url.QueryEscape("Ошибка удаления ресурса"), http.StatusSeeOther)
		return
	}

	// Log audit
	database.SafeExec(
		`INSERT INTO audit_logs (action, user_id, entity_type, entity_id, new_values, ip_address) 
		 VALUES (?, ?, 'resource', ?, ?, ?)`,
		"DELETE", user.ID, resourceIDStr, `{"is_active":false}`, r.RemoteAddr,
	)

	http.Redirect(w, r, "/admin/audit?success="+url.QueryEscape("Ресурс успешно удален"), http.StatusSeeOther)
}

// DeleteBookingAdminHandler удаляет бронирование (только админ)
func DeleteBookingAdminHandler(w http.ResponseWriter, r *http.Request) {
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

	bookingIDStr := r.FormValue("booking_id")
	if bookingIDStr == "" {
		http.Redirect(w, r, "/admin/audit?error="+url.QueryEscape("ID бронирования не указан"), http.StatusSeeOther)
		return
	}

	// Cancel booking
	_, err = database.SafeExec(
		"UPDATE bookings SET status = 'cancelled', cancellation_reason = 'Deleted by admin', updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		bookingIDStr,
	)
	if err != nil {
		http.Redirect(w, r, "/admin/audit?error="+url.QueryEscape("Ошибка удаления бронирования"), http.StatusSeeOther)
		return
	}

	// Log audit
	database.SafeExec(
		`INSERT INTO audit_logs (action, user_id, entity_type, entity_id, new_values, ip_address) 
		 VALUES (?, ?, 'booking', ?, ?, ?)`,
		"DELETE", user.ID, bookingIDStr, `{"status":"cancelled"}`, r.RemoteAddr,
	)

	http.Redirect(w, r, "/admin/audit?success="+url.QueryEscape("Бронирование успешно удалено"), http.StatusSeeOther)
}

// FormatJSONValues форматирует JSON значения для отображения
func FormatJSONValues(jsonStr string) string {
	if jsonStr == "" {
		return ""
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	formatted, _ := json.MarshalIndent(data, "", "  ")
	return string(formatted)
}
