package resource

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/auth"
)

// EditResourceHandler показывает страницу редактирования ресурса (админ или создатель)
func EditResourceHandler(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// получаем id ресурса из запроса
	resourceIDStr := r.URL.Query().Get("id")
	resourceID, err := strconv.Atoi(resourceIDStr)
	if err != nil {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Invalid resource ID"), http.StatusSeeOther)
		return
	}

	// получаем данные ресурса
	var res Resource
	var createdBy int
	var images sql.NullString
	var location sql.NullString
	var salePrice sql.NullFloat64
	var bookingType sql.NullString
	
	err = database.SafeQueryRow(
		"SELECT id, name, description, category, location, base_price, sale_price, booking_type, created_by, images FROM resources WHERE id = ?",
		resourceID,
	).Scan(&res.ID, &res.Name, &res.Description, &res.Category, &location, &res.BasePrice, &salePrice, &bookingType, &createdBy, &images)

	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Resource not found"), http.StatusSeeOther)
		return
	}

	res.CreatedBy = createdBy

	// Set defaults
	if location.Valid {
		res.Location = location.String
	} else {
		res.Location = "Astana"
	}
	
	if salePrice.Valid {
		res.SalePrice = salePrice.Float64
	}
	
	if bookingType.Valid {
		res.BookingType = bookingType.String
	} else {
		res.BookingType = "booking"
	}

	// Parse existing images
	if images.Valid && images.String != "" {
		json.Unmarshal([]byte(images.String), &res.Images)
	}

	// проверяем права доступа - только админ или создатель
	if createdBy != user.ID && user.Role != "admin" {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("You don't have access to this resource"), http.StatusSeeOther)
		return
	}

	errorMsg := r.URL.Query().Get("error")
	successMsg := r.URL.Query().Get("success")

	data := map[string]interface{}{
		"User":      user,
		"Resource":  res,
		"Locations": GetKazakhstanCities(),
		"Error":     errorMsg,
		"Success":   successMsg,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/edit_resource.html"))
	tmpl.ExecuteTemplate(w, "layout.html", data)
}

// UpdateResourceFormHandler обновляет ресурс (HTML форма)
func UpdateResourceFormHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := auth.GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// парсим форму - сначала пробуем multipart (для файлов), потом обычную
	contentType := r.Header.Get("Content-Type")
	if contentType != "" && (contentType == "multipart/form-data" || len(contentType) > 19 && contentType[:19] == "multipart/form-data") {
		err = r.ParseMultipartForm(32 << 20) // максимум 32MB
		if err != nil {
			log.Printf("Error parsing multipart form in UpdateResourceFormHandler: %v", err)
			http.Redirect(w, r, "/resources/edit?error="+url.QueryEscape("Error processing form: "+err.Error()), http.StatusSeeOther)
			return
		}
	} else {
		// обычная форма без файлов
		err = r.ParseForm()
		if err != nil {
			log.Printf("Error parsing form in UpdateResourceFormHandler: %v", err)
			http.Redirect(w, r, "/resources/edit?error="+url.QueryEscape("Error processing form: "+err.Error()), http.StatusSeeOther)
			return
		}
	}

	resourceIDStr := r.FormValue("resource_id")
	resourceID, err := strconv.Atoi(resourceIDStr)
	if err != nil {
		http.Redirect(w, r, "/resources/edit?id="+resourceIDStr+"&error="+url.QueryEscape("Invalid ID"), http.StatusSeeOther)
		return
	}

	// проверяем права доступа
	var createdBy int
	err = database.SafeQueryRow(
		"SELECT created_by FROM resources WHERE id = ?",
		resourceID,
	).Scan(&createdBy)

	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Resource not found"), http.StatusSeeOther)
		return
	}

	if createdBy != user.ID && user.Role != "admin" {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("You don't have permission to edit"), http.StatusSeeOther)
		return
	}

	// Get form values
	name := r.FormValue("name")
	description := r.FormValue("description")
	category := r.FormValue("category")
	location := r.FormValue("location")
	bookingType := r.FormValue("booking_type")
	priceStr := r.FormValue("base_price")
	salePriceStr := r.FormValue("sale_price")
	isActiveStr := r.FormValue("is_active")

	if location == "" {
		location = "Astana"
	}

	// Build update query
	updates := []string{}
	args := []interface{}{}

	if name != "" {
		updates = append(updates, "name = ?")
		args = append(args, name)
	}

	if description != "" {
		updates = append(updates, "description = ?")
		args = append(args, description)
	}

	if category != "" {
		updates = append(updates, "category = ?")
		args = append(args, category)
	}

	if location != "" {
		updates = append(updates, "location = ?")
		args = append(args, location)
	}

	if bookingType != "" {
		updates = append(updates, "booking_type = ?")
		args = append(args, bookingType)
	}

	if priceStr != "" {
		price, err := strconv.ParseFloat(priceStr, 64)
		if err == nil {
			updates = append(updates, "base_price = ?")
			args = append(args, price)
		}
	}

	if salePriceStr != "" {
		salePrice, err := strconv.ParseFloat(salePriceStr, 64)
		if err == nil {
			updates = append(updates, "sale_price = ?")
			args = append(args, salePrice)
		}
	}

	if isActiveStr != "" {
		isActive := isActiveStr == "1"
		updates = append(updates, "is_active = ?")
		args = append(args, isActive)
	}

	// проверяем что есть изменения
	if len(updates) == 0 {
		http.Redirect(w, r, "/resources/edit?id="+resourceIDStr+"&error="+url.QueryEscape("No changes to save"), http.StatusSeeOther)
		return
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, resourceID)

	// обновляем ресурс в бд
	query := "UPDATE resources SET " + strings.Join(updates, ", ") + " WHERE id = ?"
	_, err = database.SafeExec(query, args...)
	if err != nil {
		http.Redirect(w, r, "/resources/edit?id="+resourceIDStr+"&error="+url.QueryEscape("Error updating resource"), http.StatusSeeOther)
		return
	}

	// обрабатываем загрузку изображений если есть
	newImages, uploadErr := UploadResourceImages(resourceID, r)
	if uploadErr == nil && len(newImages) > 0 {
		// получаем существующие изображения
		var existingImages sql.NullString
		database.SafeQueryRow("SELECT images FROM resources WHERE id = ?", resourceID).Scan(&existingImages)
		
		var imageList []string
		if existingImages.Valid && existingImages.String != "" {
			json.Unmarshal([]byte(existingImages.String), &imageList)
		}
		
		// добавляем новые изображения
		imageList = append(imageList, newImages...)
		
		// сохраняем обновленный список
		saveErr := SaveResourceImages(resourceID, imageList)
		if saveErr != nil {
			http.Redirect(w, r, "/resources/edit?id="+resourceIDStr+"&error="+url.QueryEscape("Error saving images: "+saveErr.Error()), http.StatusSeeOther)
			return
		}
	} else if uploadErr != nil {
		// логируем ошибку но не прерываем обновление
		// изображения опциональны
	}

	// логируем в аудит
	database.SafeExec(
		`INSERT INTO audit_logs (action, user_id, entity_type, entity_id, new_values, ip_address) 
		 VALUES (?, ?, 'resource', ?, ?, ?)`,
		"UPDATE", user.ID, resourceID, `{"name":"`+name+`"}`, r.RemoteAddr,
	)

	http.Redirect(w, r, "/resources/edit?id="+resourceIDStr+"&success="+url.QueryEscape("Resource updated successfully"), http.StatusSeeOther)
}
