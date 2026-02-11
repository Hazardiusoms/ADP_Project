package resource

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/auth"
)

// ViewResourceHandler показывает детальную страницу ресурса
func ViewResourceHandler(w http.ResponseWriter, r *http.Request) {
	// получаем пользователя (может быть nil для публичного просмотра)
	user, _ := auth.GetCurrentUser(r)

	// получаем id ресурса из query
	resourceIDStr := r.URL.Query().Get("id")
	resourceID, err := strconv.Atoi(resourceIDStr)
	if err != nil {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Invalid resource ID"), http.StatusSeeOther)
		return
	}

	// получаем данные ресурса из бд
	var res Resource
	var images sql.NullString
	var location sql.NullString
	var salePrice sql.NullFloat64
	var bookingType sql.NullString
	
	err = database.SafeQueryRow(
		"SELECT id, name, description, category, location, base_price, sale_price, booking_type, created_by, images FROM resources WHERE id = ? AND is_active = 1",
		resourceID,
	).Scan(&res.ID, &res.Name, &res.Description, &res.Category, &location, &res.BasePrice, &salePrice, &bookingType, &res.CreatedBy, &images)

	// если ресурс не найден
	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Resource not found"), http.StatusSeeOther)
		return
	}

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

	// получаем отзывы для ресурса
	reviews, _ := GetReviewsForResource(resourceID)

	// проверяем, оставлял ли текущий пользователь отзыв
	var userReview *Review
	if user != nil {
		var reviewID int
		var reviewRating int
		var reviewComment string
		err = database.SafeQueryRow(
			"SELECT id, rating, comment FROM reviews WHERE resource_id = ? AND user_id = ?",
			resourceID, user.ID,
		).Scan(&reviewID, &reviewRating, &reviewComment)
		if err == nil {
			userReview = &Review{
				ID:      reviewID,
				Rating:  reviewRating,
				Comment: reviewComment,
			}
		}
	}

	errorMsg := r.URL.Query().Get("error")
	successMsg := r.URL.Query().Get("success")

	data := map[string]interface{}{
		"User":       user,
		"Resource":   res,
		"Reviews":    reviews,
		"UserReview": userReview,
		"Error":      errorMsg,
		"Success":    successMsg,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/view_resource.html"))
	tmpl.ExecuteTemplate(w, "layout.html", data)
}
