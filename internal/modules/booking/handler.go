package booking

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
	"github.com/Hazardiusoms/ADP_Project/internal/models"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/auth"
	"github.com/Hazardiusoms/ADP_Project/internal/services"
)

// CreateBookingHandler обрабатывает создание бронирования (форма и json)
func CreateBookingHandler(w http.ResponseWriter, r *http.Request) {
	// получаем текущего пользователя
	user, err := auth.GetCurrentUser(r)
	if err != nil {
		// если json запрос - возвращаем json ошибку
		if r.Header.Get("Content-Type") == "application/json" {
			respondJSON(w, http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Error:   "Unauthorized",
			})
			return
		}
		// иначе редирект на логин
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// если get запрос - показываем форму
	if r.Method == "GET" {
		// получаем список ресурсов для выбора
		rows, _ := database.SafeQuery("SELECT id, name, category, location, base_price, sale_price, booking_type FROM resources WHERE is_active = 1")
		defer rows.Close()
		var resources []map[string]interface{}
		for rows.Next() {
			var id int
			var name, category, location, bookingType string
			var price, salePrice float64
			var locationNull, bookingTypeNull sql.NullString
			var salePriceNull sql.NullFloat64
			
			// сканируем данные
			rows.Scan(&id, &name, &category, &locationNull, &price, &salePriceNull, &bookingTypeNull)
			
			// обрабатываем nullable поля
			if locationNull.Valid {
				location = locationNull.String
			} else {
				location = "Astana"
			}
			
			if bookingTypeNull.Valid {
				bookingType = bookingTypeNull.String
			} else {
				bookingType = "booking"
			}
			
			if salePriceNull.Valid {
				salePrice = salePriceNull.Float64
			}
			
			// добавляем в список
			resources = append(resources, map[string]interface{}{
				"ID":          id,
				"Name":        name,
				"Category":    category,
				"Location":    location,
				"Price":       price,
				"SalePrice":   salePrice,
				"BookingType": bookingType,
			})
		}

		// получаем предвыбранный resource_id из query если есть
		preselectedResourceID := r.URL.Query().Get("resource_id")

		// получаем сообщения об ошибках/успехе из url
		errorMsg := r.URL.Query().Get("error")
		successMsg := r.URL.Query().Get("success")

		data := map[string]interface{}{
			"User":                 user,
			"Resources":            resources,
			"PreselectedResourceID": preselectedResourceID,
			"Error":                errorMsg,
			"Success":              successMsg,
		}
		tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/create_booking.html"))
		tmpl.ExecuteTemplate(w, "layout.html", data)
		return
	}

		// проверяем если это json запрос
		if r.Header.Get("Content-Type") == "application/json" {
			handleJSONCreateBooking(w, r, user.ID)
			return
		}

	// парсим форму
	if err := SafeParseForm(r); err != nil {
		log.Printf("Error parsing form in CreateBookingHandler: %v", err)
		http.Redirect(w, r, "/bookings/create?error="+url.QueryEscape("Error processing form: "+err.Error()), http.StatusSeeOther)
		return
	}

	resourceID, _ := strconv.Atoi(r.FormValue("resource_id"))
	startTimeStr := r.FormValue("start_time")
	endTimeStr := r.FormValue("end_time")

	// проверяем что ресурс выбран
	if resourceID == 0 {
		http.Redirect(w, r, "/bookings/create?error="+url.QueryEscape("Please select a resource"), http.StatusSeeOther)
		return
	}

	// проверяем тип ресурса
	var bookingType string
	var salePrice sql.NullFloat64
	err = database.SafeQueryRow(
		"SELECT booking_type, sale_price FROM resources WHERE id = ? AND is_active = 1",
		resourceID,
	).Scan(&bookingType, &salePrice)

	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/bookings/create?error="+url.QueryEscape("Resource not found"), http.StatusSeeOther)
		return
	}

	if bookingType == "" {
		bookingType = "booking"
	}

	var booking *models.Booking
	var notificationType, notificationSubject, notificationContent, successMessage string

	// обрабатываем в зависимости от типа
	if bookingType == "sale" {
		// продажа - время не нужно, фиксированная цена
		if !salePrice.Valid || salePrice.Float64 <= 0 {
			http.Redirect(w, r, "/bookings/create?error="+url.QueryEscape("Sale price not specified"), http.StatusSeeOther)
			return
		}

		booking, err = createSale(user.ID, resourceID, salePrice.Float64, r.RemoteAddr)
		if err != nil {
			http.Redirect(w, r, "/bookings/create?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
			return
		}

		notificationType = "purchase_confirmation"
		notificationSubject = "Purchase Confirmed"
		notificationContent = "Your purchase has been completed successfully. Booking ID: " + strconv.Itoa(booking.ID)
		successMessage = "Purchase completed successfully!"
	} else {
		// бронирование - время обязательно
		if startTimeStr == "" || endTimeStr == "" {
			http.Redirect(w, r, "/bookings/create?error="+url.QueryEscape("Please specify start and end time"), http.StatusSeeOther)
			return
		}

		startTime, err := time.Parse("2006-01-02T15:04", startTimeStr)
		if err != nil {
			http.Redirect(w, r, "/bookings/create?error="+url.QueryEscape("Invalid start time format"), http.StatusSeeOther)
			return
		}

		endTime, err := time.Parse("2006-01-02T15:04", endTimeStr)
		if err != nil {
			http.Redirect(w, r, "/bookings/create?error="+url.QueryEscape("Invalid end time format"), http.StatusSeeOther)
			return
		}

		booking, err = createBooking(user.ID, resourceID, startTime, endTime, r.RemoteAddr)
		if err != nil {
			http.Redirect(w, r, "/bookings/create?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
			return
		}

		notificationType = "booking_confirmation"
		notificationSubject = "Booking Confirmed"
		notificationContent = "Your booking has been created successfully. Booking ID: " + strconv.Itoa(booking.ID)
		successMessage = "Booking created successfully!"
	}

	// отправляем уведомление асинхронно
	services.NotificationChan <- services.NotificationTask{
		UserID:  user.ID,
		Type:    notificationType,
		Subject: notificationSubject,
		Content: notificationContent,
	}

	http.Redirect(w, r, "/dashboard?success="+url.QueryEscape(successMessage), http.StatusSeeOther)
}

// handleJSONCreateBooking handles JSON booking creation
func handleJSONCreateBooking(w http.ResponseWriter, r *http.Request, userID int) {
	var req models.CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if req.ResourceID == 0 || req.StartTime.IsZero() || req.EndTime.IsZero() {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Missing required fields",
		})
		return
	}

	booking, err := createBooking(userID, req.ResourceID, req.StartTime, req.EndTime, r.RemoteAddr)
	if err != nil {
		respondJSON(w, http.StatusConflict, models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// отправляем уведомление асинхронно
	services.NotificationChan <- services.NotificationTask{
		UserID:  userID,
		Type:    "booking_confirmation",
		Subject: "Booking Confirmed",
		Content: "Your booking has been created successfully. Booking ID: " + strconv.Itoa(booking.ID),
	}

	respondJSON(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "Booking created successfully",
		Data:    booking,
	})
}

// createSale creates a sale booking (no time, fixed price)
func createSale(userID, resourceID int, salePrice float64, ipAddress string) (*models.Booking, error) {
	// Create sale booking with fixed price
	result, err := database.SafeExec(
		`INSERT INTO bookings (user_id, resource_id, start_time, end_time, status, total_price, created_by) 
		 VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'confirmed', ?, ?)`,
		userID, resourceID, salePrice, userID,
	)
	if err != nil {
		return nil, &BookingError{Message: "Error creating sale"}
	}

	bookingID, _ := result.LastInsertId()

	// Create payment record
	database.SafeExec(
		`INSERT INTO payments (booking_id, payment_method, amount, currency, status) 
		 VALUES (?, 'pending', ?, 'USD', 'pending')`,
		bookingID, salePrice,
	)

	// Fetch created booking
	var booking models.Booking
	err = database.SafeQueryRow(
		`SELECT id, user_id, resource_id, start_time, end_time, status, total_price, created_at 
		 FROM bookings WHERE id = ?`,
		bookingID,
	).Scan(&booking.ID, &booking.UserID, &booking.ResourceID, &booking.StartTime, &booking.EndTime, &booking.Status, &booking.TotalPrice, &booking.CreatedAt)

	if err != nil {
		return nil, &BookingError{Message: "Error fetching created sale"}
	}

	// Log audit
	database.SafeExec(
		`INSERT INTO audit_logs (action, user_id, entity_type, entity_id, new_values, ip_address) 
		 VALUES (?, ?, 'booking', ?, ?, ?)`,
		"CREATE_SALE", userID, booking.ID, `{"status":"confirmed","type":"sale"}`, ipAddress,
	)

	return &booking, nil
}

// createBooking создает бронирование с проверкой конфликтов
func createBooking(userID, resourceID int, startTime, endTime time.Time, ipAddress string) (*models.Booking, error) {
	// проверяем что время корректное
	if endTime.Before(startTime) || endTime.Equal(startTime) {
		return nil, &BookingError{Message: "End time must be after start time"}
	}

	// проверяем на конфликты с другими бронированиями
	var count int
	checkQuery := `SELECT COUNT(*) FROM bookings 
		WHERE resource_id = ? AND status != 'cancelled' 
		AND ((start_time < ? AND end_time > ?) OR (start_time < ? AND end_time > ?) OR (start_time >= ? AND start_time < ?))`
	err := database.SafeQueryRow(
		checkQuery,
		resourceID, endTime, startTime, endTime, startTime, startTime, endTime,
	).Scan(&count)

	// если ошибка при проверке
	if err != nil && err != sql.ErrNoRows {
		return nil, &BookingError{Message: "Error checking availability"}
	}

	// если есть конфликт
	if count > 0 {
		return nil, &BookingError{Message: "CONFLICT: Time slot is already booked"}
	}

	// получаем цену ресурса
	var basePrice float64
	err = database.SafeQueryRow(
		"SELECT base_price FROM resources WHERE id = ? AND is_active = 1",
		resourceID,
	).Scan(&basePrice)

	if err == sql.ErrNoRows {
		return nil, &BookingError{Message: "Resource not found"}
	}

	// считаем общую цену (цена за час * длительность)
	duration := endTime.Sub(startTime).Hours()
	totalPrice := basePrice * duration

	// создаем бронирование
	result, err := database.SafeExec(
		`INSERT INTO bookings (user_id, resource_id, start_time, end_time, status, total_price, created_by) 
		 VALUES (?, ?, ?, ?, 'pending', ?, ?)`,
		userID, resourceID, startTime, endTime, totalPrice, userID,
	)
	if err != nil {
		return nil, &BookingError{Message: "Error creating booking"}
	}

	// получаем id созданного бронирования
	bookingID, _ := result.LastInsertId()

	// создаем запись о платеже
	database.SafeExec(
		`INSERT INTO payments (booking_id, payment_method, amount, currency, status) 
		 VALUES (?, 'pending', ?, 'USD', 'pending')`,
		bookingID, totalPrice,
	)

	// логируем в аудит
	database.SafeExec(
		`INSERT INTO audit_logs (action, user_id, entity_type, entity_id, new_values, ip_address) 
		 VALUES (?, ?, 'booking', ?, ?, ?)`,
		"CREATE", userID, bookingID, `{"status":"pending"}`, ipAddress,
	)

	// получаем созданное бронирование из бд
	var booking models.Booking
	err = database.SafeQueryRow(
		`SELECT id, user_id, resource_id, start_time, end_time, status, total_price, created_at 
		 FROM bookings WHERE id = ?`,
		bookingID,
	).Scan(&booking.ID, &booking.UserID, &booking.ResourceID, &booking.StartTime, &booking.EndTime, &booking.Status, &booking.TotalPrice, &booking.CreatedAt)

	if err != nil {
		return nil, &BookingError{Message: "Error fetching created booking"}
	}

	return &booking, nil
}

// GetBookingsHandler возвращает все бронирования текущего пользователя (json api)
func GetBookingsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetCurrentUser(r)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	rows, err := database.SafeQuery(
		`SELECT id, user_id, resource_id, start_time, end_time, status, total_price, created_at 
		 FROM bookings WHERE user_id = ? ORDER BY created_at DESC`,
		user.ID,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Error fetching bookings",
		})
		return
	}
	defer rows.Close()

	var bookings []models.Booking
	for rows.Next() {
		var booking models.Booking
		err := rows.Scan(&booking.ID, &booking.UserID, &booking.ResourceID, &booking.StartTime, &booking.EndTime, &booking.Status, &booking.TotalPrice, &booking.CreatedAt)
		if err != nil {
			continue
		}
		bookings = append(bookings, booking)
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    bookings,
	})
}

// CancelBookingHandler отменяет бронирование (json api)
func CancelBookingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" && r.Method != "DELETE" {
		respondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	user, err := auth.GetCurrentUser(r)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid booking ID",
		})
		return
	}

		// проверяем владельца
		var bookingUserID int
	err = database.SafeQueryRow(
		"SELECT user_id FROM bookings WHERE id = ?",
		id,
	).Scan(&bookingUserID)

	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "Booking not found",
		})
		return
	}

	if bookingUserID != user.ID && user.Role != "admin" {
		respondJSON(w, http.StatusForbidden, models.APIResponse{
			Success: false,
			Error:   "Not authorized to cancel this booking",
		})
		return
	}

		// отменяем бронирование
		_, err = database.SafeExec(
		"UPDATE bookings SET status = 'cancelled', cancellation_reason = 'User cancelled', updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		id,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Error cancelling booking",
		})
		return
	}

	// Log audit
	database.SafeExec(
		`INSERT INTO audit_logs (action, user_id, entity_type, entity_id, old_values, new_values, ip_address) 
		 VALUES (?, ?, 'booking', ?, ?, ?, ?)`,
		"CANCEL", user.ID, id, `{"status":"pending"}`, `{"status":"cancelled"}`, r.RemoteAddr,
	)

		// отправляем уведомление
		services.NotificationChan <- services.NotificationTask{
		UserID:  user.ID,
		Type:    "cancellation",
		Subject: "Booking Cancelled",
		Content: "Your booking #" + idStr + " has been cancelled.",
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Booking cancelled successfully",
	})
}

// BookingError представляет ошибку связанную с бронированием
type BookingError struct {
	Message string
}

func (e *BookingError) Error() string {
	return e.Message
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, statusCode int, response models.APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
