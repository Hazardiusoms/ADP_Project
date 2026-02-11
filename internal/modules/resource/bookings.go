package resource

import (
	"database/sql"
	"html/template"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/auth"
)

// ResourceBookingInfo содержит информацию о бронировании с контактами пользователя
type ResourceBookingInfo struct {
	BookingID    int
	UserName     string
	UserEmail    string
	UserPhone    string
	StartTime    string
	EndTime      string
	Status       string
	TotalPrice   float64
	CreatedAt    string
}

// ViewResourceBookingsHandler показывает все бронирования для конкретного ресурса
func ViewResourceBookingsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	resourceIDStr := r.URL.Query().Get("resource_id")
	resourceID, err := strconv.Atoi(resourceIDStr)
	if err != nil {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Неверный ID ресурса"), http.StatusSeeOther)
		return
	}

	// Проверяем, что пользователь является создателем ресурса или админом
	var createdBy int
	err = database.SafeQueryRow(
		"SELECT created_by FROM resources WHERE id = ?",
		resourceID,
	).Scan(&createdBy)

	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Ресурс не найден"), http.StatusSeeOther)
		return
	}

	if createdBy != user.ID && user.Role != "admin" {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("У вас нет доступа к этому ресурсу"), http.StatusSeeOther)
		return
	}

	// Получаем информацию о ресурсе
	var resourceName string
	err = database.SafeQueryRow(
		"SELECT name FROM resources WHERE id = ?",
		resourceID,
	).Scan(&resourceName)

	// Получаем все бронирования для этого ресурса с контактами пользователей
	rows, err := database.SafeQuery(
		`SELECT 
			b.id, 
			u.full_name, 
			u.email, 
			u.phone, 
			b.start_time, 
			b.end_time, 
			b.status, 
			b.total_price, 
			b.created_at
		FROM bookings b
		JOIN users u ON b.user_id = u.id
		WHERE b.resource_id = ? AND b.status != 'cancelled'
		ORDER BY b.start_time ASC`,
		resourceID,
	)
	if err != nil {
		http.Error(w, "Ошибка получения бронирований", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var bookings []ResourceBookingInfo
	for rows.Next() {
		var booking ResourceBookingInfo
		var startTime, endTime, createdAt sql.NullString
		err := rows.Scan(
			&booking.BookingID,
			&booking.UserName,
			&booking.UserEmail,
			&booking.UserPhone,
			&startTime,
			&endTime,
			&booking.Status,
			&booking.TotalPrice,
			&createdAt,
		)
		if err != nil {
			continue
		}
		if startTime.Valid {
			booking.StartTime = startTime.String
		}
		if endTime.Valid {
			booking.EndTime = endTime.String
		}
		if createdAt.Valid {
			booking.CreatedAt = createdAt.String
		}
		bookings = append(bookings, booking)
	}

	data := map[string]interface{}{
		"User":         user,
		"ResourceID":   resourceID,
		"ResourceName": resourceName,
		"Bookings":     bookings,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/resource_bookings.html"))
	tmpl.ExecuteTemplate(w, "layout.html", data)
}
