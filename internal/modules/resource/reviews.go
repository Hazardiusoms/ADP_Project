package resource

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/auth"
)

// Review структура для отзыва
type Review struct {
	ID        int
	ResourceID int
	UserID   int
	UserName string
	Rating   int
	Comment  string
	CreatedAt string
}

// CreateReviewHandler создает новый отзыв
func CreateReviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := auth.GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	resourceIDStr := r.FormValue("resource_id")
	ratingStr := r.FormValue("rating")
	comment := r.FormValue("comment")

	resourceID, err := strconv.Atoi(resourceIDStr)
	if err != nil {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Invalid resource ID"), http.StatusSeeOther)
		return
	}

	rating, err := strconv.Atoi(ratingStr)
	if err != nil || rating < 1 || rating > 5 {
		rating = 5
	}

	if comment == "" {
		http.Redirect(w, r, "/resources/view?id="+resourceIDStr+"&error="+url.QueryEscape("Comment cannot be empty"), http.StatusSeeOther)
		return
	}

	// проверяем, не оставлял ли уже пользователь отзыв на этот ресурс
	var existingReviewID int
	err = database.SafeQueryRow(
		"SELECT id FROM reviews WHERE resource_id = ? AND user_id = ?",
		resourceID, user.ID,
	).Scan(&existingReviewID)

	if err == nil {
		// обновляем существующий отзыв
		database.SafeExec(
			"UPDATE reviews SET rating = ?, comment = ?, created_at = CURRENT_TIMESTAMP WHERE id = ?",
			rating, comment, existingReviewID,
		)
	} else {
		// создаем новый отзыв
		database.SafeExec(
			"INSERT INTO reviews (resource_id, user_id, rating, comment) VALUES (?, ?, ?, ?)",
			resourceID, user.ID, rating, comment,
		)
	}

	http.Redirect(w, r, "/resources/view?id="+resourceIDStr+"&success="+url.QueryEscape("Review submitted successfully"), http.StatusSeeOther)
}

// GetReviewsForResource получает все отзывы для ресурса
func GetReviewsForResource(resourceID int) ([]Review, error) {
	// получаем все отзывы с именами пользователей
	rows, err := database.SafeQuery(
		`SELECT r.id, r.resource_id, r.user_id, u.full_name, r.rating, r.comment, r.created_at 
		 FROM reviews r
		 JOIN users u ON r.user_id = u.id
		 WHERE r.resource_id = ?
		 ORDER BY r.created_at DESC`,
		resourceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// собираем отзывы в массив
	var reviews []Review
	for rows.Next() {
		var review Review
		var createdAt string
		err := rows.Scan(&review.ID, &review.ResourceID, &review.UserID, &review.UserName, &review.Rating, &review.Comment, &createdAt)
		if err != nil {
			continue
		}
		review.CreatedAt = createdAt
		reviews = append(reviews, review)
	}

	return reviews, nil
}
