package resource

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
	"github.com/Hazardiusoms/ADP_Project/internal/models"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/auth"
)

// DashboardHandler показывает дашборд с ресурсами
func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	// получаем текущего пользователя
	user, err := auth.GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// получаем параметры фильтров из url
	locationFilter := r.URL.Query().Get("location")
	categoryFilter := r.URL.Query().Get("category")
	minPriceStr := r.URL.Query().Get("min_price")
	maxPriceStr := r.URL.Query().Get("max_price")
	bookingTypeFilter := r.URL.Query().Get("booking_type")

	// строим запрос с фильтрами
	query := "SELECT id, name, description, category, location, base_price, sale_price, booking_type, created_by, images FROM resources WHERE is_active = 1"
	args := []interface{}{}
	conditions := []string{}

	// добавляем фильтр по локации если указан
	if locationFilter != "" {
		conditions = append(conditions, "location = ?")
		args = append(args, locationFilter)
	}

	// добавляем фильтр по категории если указан
	if categoryFilter != "" {
		conditions = append(conditions, "category = ?")
		args = append(args, categoryFilter)
	}

	// добавляем фильтр по типу если указан
	if bookingTypeFilter != "" {
		conditions = append(conditions, "booking_type = ?")
		args = append(args, bookingTypeFilter)
	}

	// добавляем фильтр по минимальной цене
	if minPriceStr != "" {
		if minPrice, err := strconv.ParseFloat(minPriceStr, 64); err == nil {
			conditions = append(conditions, "(base_price >= ? OR sale_price >= ?)")
			args = append(args, minPrice, minPrice)
		}
	}

	// добавляем фильтр по максимальной цене
	if maxPriceStr != "" {
		if maxPrice, err := strconv.ParseFloat(maxPriceStr, 64); err == nil {
			conditions = append(conditions, "(base_price <= ? OR sale_price <= ?)")
			args = append(args, maxPrice, maxPrice)
		}
	}

	// добавляем условия к запросу если есть
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	// сортируем по дате создания
	query += " ORDER BY created_at DESC"

	// выполняем запрос
	rows, err := database.SafeQuery(query, args...)
	if err != nil {
		http.Error(w, "Error fetching resources", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// собираем ресурсы в массив
	var resources []Resource
	for rows.Next() {
		var res Resource
		var images sql.NullString
		var location sql.NullString
		var salePrice sql.NullFloat64
		var bookingType sql.NullString
		
		// сканируем данные из строки
		rows.Scan(&res.ID, &res.Name, &res.Description, &res.Category, &location, &res.BasePrice, &salePrice, &bookingType, &res.CreatedBy, &images)
		
		// устанавливаем значения по умолчанию
		if location.Valid {
			res.Location = location.String
		} else {
			res.Location = "Astana"
		}
		
		// парсим цену продажи если есть
		if salePrice.Valid {
			res.SalePrice = salePrice.Float64
		}
		
		// парсим тип бронирования
		if bookingType.Valid {
			res.BookingType = bookingType.String
		} else {
			res.BookingType = "booking"
		}
		
		// парсим json с изображениями
		if images.Valid && images.String != "" {
			json.Unmarshal([]byte(images.String), &res.Images)
		}
		
		resources = append(resources, res)
	}

		// получаем сообщения об ошибках/успехе из url
		errorMsg := r.URL.Query().Get("error")
		successMsg := r.URL.Query().Get("success")

	// получаем уникальные локации и категории для фильтров
	locationRows, _ := database.SafeQuery("SELECT DISTINCT location FROM resources WHERE is_active = 1 AND location IS NOT NULL ORDER BY location")
	defer locationRows.Close()
	var locations []string
	for locationRows.Next() {
		var loc string
		if locationRows.Scan(&loc) == nil && loc != "" {
			locations = append(locations, loc)
		}
	}
	if len(locations) == 0 {
		locations = GetKazakhstanCities()
	}

	data := map[string]interface{}{
		"User":            user,
		"Resources":       resources,
		"Locations":       locations,
		"Categories":      []string{"room", "equipment", "service", "other"},
		"BookingTypes":    []string{"booking", "sale"},
		"CurrentLocation": locationFilter,
		"CurrentCategory": categoryFilter,
		"CurrentType":     bookingTypeFilter,
		"MinPrice":        minPriceStr,
		"MaxPrice":        maxPriceStr,
		"Error":           errorMsg,
		"Success":         successMsg,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/dashboard.html"))
	tmpl.ExecuteTemplate(w, "layout.html", data)
}

// Resource структура для рендеринга шаблонов
type Resource struct {
	ID          int
	Name        string
	Description string
	Category    string
	Location    string
	BasePrice   float64
	SalePrice   float64
	BookingType string
	CreatedBy   int
	Images      []string
}

// CreateResourceHandler обрабатывает создание ресурса (форма и json)
func CreateResourceHandler(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetCurrentUser(r)
	if err != nil {
		if r.Header.Get("Content-Type") == "application/json" {
			respondJSON(w, http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Error:   "Unauthorized",
			})
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == "GET" {
		// получаем сообщения об ошибках/успехе из url
		errorMsg := r.URL.Query().Get("error")
		successMsg := r.URL.Query().Get("success")
		
		data := map[string]interface{}{
			"User":      user,
			"Locations": GetKazakhstanCities(),
			"Error":     errorMsg,
			"Success":   successMsg,
		}
		
		tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/create_resource.html"))
		tmpl.ExecuteTemplate(w, "layout.html", data)
		return
	}

	// проверяем если это json запрос
	if r.Header.Get("Content-Type") == "application/json" {
		handleJSONCreateResource(w, r, user.ID)
		return
	}

	// парсим multipart форму для загрузки файлов
	err = r.ParseMultipartForm(32 << 20) // максимум 32MB
	if err != nil {
		http.Redirect(w, r, "/resources/create?error="+url.QueryEscape("Error processing form"), http.StatusSeeOther)
		return
	}

	// получаем значения из формы
	name := r.FormValue("name")
	category := r.FormValue("category")
	description := r.FormValue("description")
	location := r.FormValue("location")
	bookingType := r.FormValue("booking_type")
	priceStr := r.FormValue("base_price")
	salePriceStr := r.FormValue("sale_price")

	// проверяем обязательные поля
	if name == "" || category == "" {
		http.Redirect(w, r, "/resources/create?error="+url.QueryEscape("Please fill in all required fields (name and category)"), http.StatusSeeOther)
		return
	}

	// устанавливаем значения по умолчанию
	if location == "" {
		location = "Astana"
	}

	if bookingType == "" {
		bookingType = "booking"
	}

	var price, salePrice float64
	// обрабатываем цену в зависимости от типа
	if bookingType == "booking" {
		if priceStr == "" {
			http.Redirect(w, r, "/resources/create?error="+url.QueryEscape("Please specify price per hour"), http.StatusSeeOther)
			return
		}
		price, err = strconv.ParseFloat(priceStr, 64)
		if err != nil || price < 0 {
			http.Redirect(w, r, "/resources/create?error="+url.QueryEscape("Invalid price. Please use a positive number."), http.StatusSeeOther)
			return
		}
	} else if bookingType == "sale" {
		if salePriceStr == "" {
			http.Redirect(w, r, "/resources/create?error="+url.QueryEscape("Please specify sale price"), http.StatusSeeOther)
			return
		}
		salePrice, err = strconv.ParseFloat(salePriceStr, 64)
		if err != nil || salePrice < 0 {
			http.Redirect(w, r, "/resources/create?error="+url.QueryEscape("Invalid sale price. Please use a positive number."), http.StatusSeeOther)
			return
		}
	}

	// создаем ресурс в бд
	result, err := database.SafeExec(
		"INSERT INTO resources (name, description, category, location, base_price, sale_price, booking_type, created_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		name, description, category, location, price, salePrice, bookingType, user.ID,
	)
	if err != nil {
		http.Redirect(w, r, "/resources/create?error="+url.QueryEscape("Error creating resource. Please try again."), http.StatusSeeOther)
		return
	}

	// получаем id созданного ресурса
	resourceID, err := result.LastInsertId()
	if err != nil {
		http.Redirect(w, r, "/dashboard?success="+url.QueryEscape("Resource '"+name+"' created, but images were not uploaded"), http.StatusSeeOther)
		return
	}

	// загружаем изображения если есть
	images, err := UploadResourceImages(int(resourceID), r)
	if err == nil && len(images) > 0 {
		SaveResourceImages(int(resourceID), images)
	}

	http.Redirect(w, r, "/dashboard?success="+url.QueryEscape("Resource '"+name+"' created successfully!"), http.StatusSeeOther)
}

// handleJSONCreateResource handles JSON resource creation
func handleJSONCreateResource(w http.ResponseWriter, r *http.Request, userID int) {
	var req models.CreateResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if req.Name == "" || req.Category == "" {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Missing required fields",
		})
		return
	}

	result, err := database.SafeExec(
		"INSERT INTO resources (name, description, category, base_price, max_duration_hours, min_booking_hours, created_by) VALUES (?, ?, ?, ?, ?, ?, ?)",
		req.Name, req.Description, req.Category, req.BasePrice, req.MaxDurationHours, req.MinBookingHours, userID,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Error creating resource",
		})
		return
	}

	resourceID, _ := result.LastInsertId()
	respondJSON(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "Resource created successfully",
		Data: map[string]interface{}{
			"resource_id": resourceID,
			"name":        req.Name,
		},
	})
}

// GetResourcesHandler возвращает все ресурсы (json api)
func GetResourcesHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := database.SafeQuery(
		"SELECT id, name, description, category, base_price, max_duration_hours, min_booking_hours FROM resources WHERE is_active = 1",
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Error fetching resources",
		})
		return
	}
	defer rows.Close()

	var resources []models.Resource
	for rows.Next() {
		var res models.Resource
		err := rows.Scan(&res.ID, &res.Name, &res.Description, &res.Category, &res.BasePrice, &res.MaxDurationHours, &res.MinBookingHours)
		if err != nil {
			continue
		}
		resources = append(resources, res)
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    resources,
	})
}

// GetResourceHandler возвращает один ресурс по id (json api)
func GetResourceHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Resource ID required",
		})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid resource ID",
		})
		return
	}

	var res models.Resource
	err = database.SafeQueryRow(
		"SELECT id, name, description, category, base_price, max_duration_hours, min_booking_hours FROM resources WHERE id = ? AND is_active = 1",
		id,
	).Scan(&res.ID, &res.Name, &res.Description, &res.Category, &res.BasePrice, &res.MaxDurationHours, &res.MinBookingHours)

	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "Resource not found",
		})
		return
	}

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Error fetching resource",
		})
		return
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    res,
	})
}

// UpdateResourceHandler обновляет ресурс (json api)
func UpdateResourceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" && r.Method != "PATCH" {
		respondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid resource ID",
		})
		return
	}

	var req models.CreateResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	_, err = database.SafeExec(
		"UPDATE resources SET name = ?, description = ?, category = ?, base_price = ?, max_duration_hours = ?, min_booking_hours = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		req.Name, req.Description, req.Category, req.BasePrice, req.MaxDurationHours, req.MinBookingHours, id,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Error updating resource",
		})
		return
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Resource updated successfully",
	})
}

// DeleteResourceHandler удаляет ресурс (мягкое удаление)
func DeleteResourceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		respondJSON(w, http.StatusMethodNotAllowed, models.APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid resource ID",
		})
		return
	}

	_, err = database.SafeExec(
		"UPDATE resources SET is_active = 0, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		id,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Error deleting resource",
		})
		return
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Resource deleted successfully",
	})
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, statusCode int, response models.APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
