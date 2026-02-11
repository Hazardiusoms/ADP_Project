package booking

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
	"github.com/Hazardiusoms/ADP_Project/internal/models"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/auth"
)

// EditBookingHandler показывает страницу редактирования бронирования
func EditBookingHandler(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	bookingIDStr := r.URL.Query().Get("id")
	bookingID, err := strconv.Atoi(bookingIDStr)
	if err != nil {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Неверный ID бронирования"), http.StatusSeeOther)
		return
	}

	// Get booking with resource info
	var booking models.Booking
	var resourceName string
	var images sql.NullString
	err = database.SafeQueryRow(
		`SELECT b.id, b.user_id, b.resource_id, b.start_time, b.end_time, b.status, 
		 b.total_price, b.created_at, b.images, r.name
		 FROM bookings b
		 JOIN resources r ON b.resource_id = r.id
		 WHERE b.id = ?`,
		bookingID,
	).Scan(
		&booking.ID, &booking.UserID, &booking.ResourceID, &booking.StartTime,
		&booking.EndTime, &booking.Status, &booking.TotalPrice, &booking.CreatedAt,
		&images, &resourceName,
	)

	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Бронирование не найдено"), http.StatusSeeOther)
		return
	}

	// Check permissions: user can edit their own booking, admin can edit any
	if booking.UserID != user.ID && user.Role != "admin" {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("У вас нет доступа к этому бронированию"), http.StatusSeeOther)
		return
	}

	// Parse images
	var imageList []string
	if images.Valid && images.String != "" {
		json.Unmarshal([]byte(images.String), &imageList)
	}

	// Get error/success message from URL
	errorMsg := r.URL.Query().Get("error")
	successMsg := r.URL.Query().Get("success")

	data := map[string]interface{}{
		"User":         user,
		"Booking":      booking,
		"ResourceName": resourceName,
		"Images":       imageList,
		"Error":        errorMsg,
		"Success":      successMsg,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/edit_booking.html"))
	tmpl.ExecuteTemplate(w, "layout.html", data)
}

// UpdateBookingHandler обновляет бронирование
func UpdateBookingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := auth.GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	bookingIDStr := r.FormValue("booking_id")
	bookingID, err := strconv.Atoi(bookingIDStr)
	if err != nil {
		http.Redirect(w, r, "/bookings/edit?id="+bookingIDStr+"&error="+url.QueryEscape("Неверный ID"), http.StatusSeeOther)
		return
	}

	// Check permissions
	var bookingUserID int
	err = database.SafeQueryRow(
		"SELECT user_id FROM bookings WHERE id = ?",
		bookingID,
	).Scan(&bookingUserID)

	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Бронирование не найдено"), http.StatusSeeOther)
		return
	}

	if bookingUserID != user.ID && user.Role != "admin" {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("У вас нет прав для редактирования"), http.StatusSeeOther)
		return
	}

	// Get form values
	status := r.FormValue("status")
	startTimeStr := r.FormValue("start_time")
	endTimeStr := r.FormValue("end_time")

	// Build update query
	updates := []string{}
	args := []interface{}{}

	if status != "" {
		updates = append(updates, "status = ?")
		args = append(args, status)
	}

	if startTimeStr != "" {
		startTime, err := time.Parse("2006-01-02T15:04", startTimeStr)
		if err == nil {
			updates = append(updates, "start_time = ?")
			args = append(args, startTime)
		}
	}

	if endTimeStr != "" {
		endTime, err := time.Parse("2006-01-02T15:04", endTimeStr)
		if err == nil {
			updates = append(updates, "end_time = ?")
			args = append(args, endTime)
		}
	}

	if len(updates) == 0 {
		http.Redirect(w, r, "/bookings/edit?id="+bookingIDStr+"&error="+url.QueryEscape("Нет изменений для сохранения"), http.StatusSeeOther)
		return
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, bookingID)

	query := "UPDATE bookings SET " + strings.Join(updates, ", ") + " WHERE id = ?"
	_, err = database.SafeExec(query, args...)
	if err != nil {
		http.Redirect(w, r, "/bookings/edit?id="+bookingIDStr+"&error="+url.QueryEscape("Ошибка обновления: "+err.Error()), http.StatusSeeOther)
		return
	}

	// Log audit
	database.SafeExec(
		`INSERT INTO audit_logs (action, user_id, entity_type, entity_id, new_values, ip_address) 
		 VALUES (?, ?, 'booking', ?, ?, ?)`,
		"UPDATE", user.ID, bookingID, `{"status":"`+status+`"}`, r.RemoteAddr,
	)

	http.Redirect(w, r, "/bookings/edit?id="+bookingIDStr+"&success="+url.QueryEscape("Бронирование успешно обновлено"), http.StatusSeeOther)
}

// ConfirmBookingHandler подтверждает бронирование (для создателя бронирования)
func ConfirmBookingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := auth.GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	bookingIDStr := r.FormValue("booking_id")
	bookingID, err := strconv.Atoi(bookingIDStr)
	if err != nil {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Неверный ID"), http.StatusSeeOther)
		return
	}

	// Check that user created this booking
	var createdBy int
	err = database.SafeQueryRow(
		"SELECT created_by FROM bookings WHERE id = ?",
		bookingID,
	).Scan(&createdBy)

	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Бронирование не найдено"), http.StatusSeeOther)
		return
	}

	if createdBy != user.ID && user.Role != "admin" {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Только создатель бронирования может подтвердить его"), http.StatusSeeOther)
		return
	}

	// Update status to confirmed
	_, err = database.SafeExec(
		"UPDATE bookings SET status = 'confirmed', updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		bookingID,
	)
	if err != nil {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Ошибка подтверждения"), http.StatusSeeOther)
		return
	}

	// Log audit
	database.SafeExec(
		`INSERT INTO audit_logs (action, user_id, entity_type, entity_id, new_values, ip_address) 
		 VALUES (?, ?, 'booking', ?, ?, ?)`,
		"CONFIRM", user.ID, bookingID, `{"status":"confirmed"}`, r.RemoteAddr,
	)

	http.Redirect(w, r, "/bookings/edit?id="+bookingIDStr+"&success="+url.QueryEscape("Бронирование подтверждено!"), http.StatusSeeOther)
}

// UploadImageHandler загружает изображение для бронирования
func UploadImageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := auth.GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	bookingIDStr := r.FormValue("booking_id")
	bookingID, err := strconv.Atoi(bookingIDStr)
	if err != nil {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Неверный ID"), http.StatusSeeOther)
		return
	}

	// Check permissions
	var createdBy int
	err = database.SafeQueryRow(
		"SELECT created_by FROM bookings WHERE id = ?",
		bookingID,
	).Scan(&createdBy)

	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Бронирование не найдено"), http.StatusSeeOther)
		return
	}

	if createdBy != user.ID && user.Role != "admin" {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("У вас нет прав для загрузки изображений"), http.StatusSeeOther)
		return
	}

	// Parse multipart form (32MB max)
	err = r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Redirect(w, r, "/bookings/edit?id="+bookingIDStr+"&error="+url.QueryEscape("Ошибка загрузки файла"), http.StatusSeeOther)
		return
	}

	// Create uploads directory if it doesn't exist
	uploadDir := "./web/uploads/bookings"
	os.MkdirAll(uploadDir, 0755)

	// Get uploaded files
	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		http.Redirect(w, r, "/bookings/edit?id="+bookingIDStr+"&error="+url.QueryEscape("Не выбраны файлы"), http.StatusSeeOther)
		return
	}

	// Get existing images
	var existingImages sql.NullString
	database.SafeQueryRow("SELECT images FROM bookings WHERE id = ?", bookingID).Scan(&existingImages)
	
	var imageList []string
	if existingImages.Valid && existingImages.String != "" {
		json.Unmarshal([]byte(existingImages.String), &imageList)
	}

	// Process each file
	for _, fileHeader := range files {
		// Open file
		file, err := fileHeader.Open()
		if err != nil {
			continue
		}
		defer file.Close()

		// Validate file type
		buffer := make([]byte, 512)
		file.Read(buffer)
		file.Seek(0, 0)
		contentType := http.DetectContentType(buffer)
		if !strings.HasPrefix(contentType, "image/") {
			continue
		}

		// Generate unique filename
		ext := filepath.Ext(fileHeader.Filename)
		filename := strconv.Itoa(bookingID) + "_" + strconv.FormatInt(time.Now().UnixNano(), 10) + ext
		filePath := filepath.Join(uploadDir, filename)

		// Create file
		dst, err := os.Create(filePath)
		if err != nil {
			continue
		}
		defer dst.Close()

		// Copy file
		io.Copy(dst, file)

		// Add to image list
		imageList = append(imageList, "/uploads/bookings/"+filename)
	}

	// Save image list to database
	imagesJSON, _ := json.Marshal(imageList)
	_, err = database.SafeExec(
		"UPDATE bookings SET images = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		string(imagesJSON), bookingID,
	)
	if err != nil {
		http.Redirect(w, r, "/bookings/edit?id="+bookingIDStr+"&error="+url.QueryEscape("Ошибка сохранения изображений"), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/bookings/edit?id="+bookingIDStr+"&success="+url.QueryEscape("Изображения успешно загружены"), http.StatusSeeOther)
}

// DeleteImageHandler удаляет изображение
func DeleteImageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := auth.GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	bookingIDStr := r.FormValue("booking_id")
	imagePath := r.FormValue("image_path")

	bookingID, err := strconv.Atoi(bookingIDStr)
	if err != nil {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Неверный ID"), http.StatusSeeOther)
		return
	}

	// Check permissions
	var createdBy int
	err = database.SafeQueryRow(
		"SELECT created_by FROM bookings WHERE id = ?",
		bookingID,
	).Scan(&createdBy)

	if createdBy != user.ID && user.Role != "admin" {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("У вас нет прав"), http.StatusSeeOther)
		return
	}

	// Get existing images
	var existingImages sql.NullString
	database.SafeQueryRow("SELECT images FROM bookings WHERE id = ?", bookingID).Scan(&existingImages)
	
	var imageList []string
	if existingImages.Valid && existingImages.String != "" {
		json.Unmarshal([]byte(existingImages.String), &imageList)
	}

	// Remove image from list
	newList := []string{}
	for _, img := range imageList {
		if img != imagePath {
			newList = append(newList, img)
		}
	}

	// Delete file from disk
	if strings.HasPrefix(imagePath, "/uploads/") {
		filePath := "." + imagePath
		os.Remove(filePath)
	}

	// Update database
	imagesJSON, _ := json.Marshal(newList)
	database.SafeExec(
		"UPDATE bookings SET images = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		string(imagesJSON), bookingID,
	)

	http.Redirect(w, r, "/bookings/edit?id="+bookingIDStr+"&success="+url.QueryEscape("Изображение удалено"), http.StatusSeeOther)
}
