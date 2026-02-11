package booking

import (
	"database/sql"
	"html/template"
	"net/http"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/auth"
)

// ListBookingsHandler показывает список всех бронирований пользователя
func ListBookingsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get all bookings for user with resource info
	rows, err := database.SafeQuery(
		`SELECT 
			b.id, 
			b.resource_id, 
			b.start_time, 
			b.end_time, 
			b.status, 
			b.total_price, 
			b.created_at,
			r.name as resource_name
		FROM bookings b
		JOIN resources r ON b.resource_id = r.id
		WHERE b.user_id = ? 
		ORDER BY b.created_at DESC`,
		user.ID,
	)
	if err != nil {
		http.Error(w, "Error while getting booking data", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type BookingInfo struct {
		ID           int
		ResourceID   int
		ResourceName string
		StartTime    string
		EndTime      string
		Status       string
		TotalPrice   float64
		CreatedAt    string
	}

	var bookings []BookingInfo
	for rows.Next() {
		var booking BookingInfo
		var startTime, endTime, createdAt sql.NullString
		err := rows.Scan(
			&booking.ID,
			&booking.ResourceID,
			&startTime,
			&endTime,
			&booking.Status,
			&booking.TotalPrice,
			&createdAt,
			&booking.ResourceName,
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

	errorMsg := r.URL.Query().Get("error")
	successMsg := r.URL.Query().Get("success")

	data := map[string]interface{}{
		"User":     user,
		"Bookings": bookings,
		"Error":    errorMsg,
		"Success":  successMsg,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/my_bookings.html"))
	tmpl.ExecuteTemplate(w, "layout.html", data)
}
