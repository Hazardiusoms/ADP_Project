package notification

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
	"github.com/Hazardiusoms/ADP_Project/internal/models"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/auth"
)

// GetNotificationsHandler returns all notifications for the current user (JSON API)
func GetNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetCurrentUser(r)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	rows, err := database.SafeQuery(
		`SELECT id, user_id, type, subject, content, status, sent_at, read_at, metadata
		 FROM notifications 
		 WHERE user_id = ? 
		 ORDER BY created_at DESC`,
		user.ID,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Error fetching notifications",
		})
		return
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var notif models.Notification
		var sentAt, readAt sql.NullTime
		err := rows.Scan(
			&notif.ID, &notif.UserID, &notif.Type, &notif.Subject,
			&notif.Content, &notif.Status, &sentAt, &readAt, &notif.Metadata,
		)
		if err != nil {
			continue
		}
		if sentAt.Valid {
			notif.SentAt = &sentAt.Time
		}
		if readAt.Valid {
			notif.ReadAt = &readAt.Time
		}
		notifications = append(notifications, notif)
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    notifications,
	})
}

// GetNotificationHandler returns a single notification by ID (JSON API)
func GetNotificationHandler(w http.ResponseWriter, r *http.Request) {
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
			Error:   "Invalid notification ID",
		})
		return
	}

	var notif models.Notification
	var sentAt, readAt sql.NullTime
	err = database.SafeQueryRow(
		`SELECT id, user_id, type, subject, content, status, sent_at, read_at, metadata
		 FROM notifications WHERE id = ? AND user_id = ?`,
		id, user.ID,
	).Scan(
		&notif.ID, &notif.UserID, &notif.Type, &notif.Subject,
		&notif.Content, &notif.Status, &sentAt, &readAt, &notif.Metadata,
	)

	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "Notification not found",
		})
		return
	}

	if sentAt.Valid {
		notif.SentAt = &sentAt.Time
	}
	if readAt.Valid {
		notif.ReadAt = &readAt.Time
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    notif,
	})
}

// CreateNotificationHandler creates a new notification (JSON API)
func CreateNotificationHandler(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetCurrentUser(r)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	// Only admin can create notifications
	if user.Role != "admin" {
		respondJSON(w, http.StatusForbidden, models.APIResponse{
			Success: false,
			Error:   "Forbidden: Admin access required",
		})
		return
	}

	var req struct {
		UserID  int    `json:"user_id"`
		Type    string `json:"type"`
		Subject string `json:"subject"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if req.UserID == 0 || req.Type == "" || req.Content == "" {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Missing required fields",
		})
		return
	}

	result, err := database.SafeExec(
		`INSERT INTO notifications (user_id, type, subject, content, status) 
		 VALUES (?, ?, ?, ?, 'pending')`,
		req.UserID, req.Type, req.Subject, req.Content,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Error creating notification",
		})
		return
	}

	notificationID, _ := result.LastInsertId()
	respondJSON(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "Notification created successfully",
		Data: map[string]interface{}{
			"notification_id": notificationID,
		},
	})
}

// UpdateNotificationHandler updates a notification (mark as read, etc.) (JSON API)
func UpdateNotificationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" && r.Method != "PATCH" {
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
			Error:   "Invalid notification ID",
		})
		return
	}

	var req struct {
		Status string `json:"status"`
		Read   bool   `json:"read"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	// Verify ownership
	var notificationUserID int
	err = database.SafeQueryRow(
		"SELECT user_id FROM notifications WHERE id = ?",
		id,
	).Scan(&notificationUserID)

	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "Notification not found",
		})
		return
	}

	if notificationUserID != user.ID && user.Role != "admin" {
		respondJSON(w, http.StatusForbidden, models.APIResponse{
			Success: false,
			Error:   "Not authorized to update this notification",
		})
		return
	}

	// Update notification
	updateQuery := "UPDATE notifications SET"
	args := []interface{}{}
	
	if req.Status != "" {
		updateQuery += " status = ?"
		args = append(args, req.Status)
	}
	
	if req.Read {
		if len(args) > 0 {
			updateQuery += ","
		}
		updateQuery += " read_at = CURRENT_TIMESTAMP"
	}
	
	updateQuery += " WHERE id = ? AND user_id = ?"
	args = append(args, id, notificationUserID)

	_, err = database.SafeExec(updateQuery, args...)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Error updating notification",
		})
		return
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Notification updated successfully",
	})
}

// DeleteNotificationHandler deletes a notification (JSON API)
func DeleteNotificationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
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
			Error:   "Invalid notification ID",
		})
		return
	}

	// Verify ownership
	var notificationUserID int
	err = database.SafeQueryRow(
		"SELECT user_id FROM notifications WHERE id = ?",
		id,
	).Scan(&notificationUserID)

	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "Notification not found",
		})
		return
	}

	if notificationUserID != user.ID && user.Role != "admin" {
		respondJSON(w, http.StatusForbidden, models.APIResponse{
			Success: false,
			Error:   "Not authorized to delete this notification",
		})
		return
	}

	_, err = database.SafeExec(
		"DELETE FROM notifications WHERE id = ? AND user_id = ?",
		id, notificationUserID,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Error deleting notification",
		})
		return
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Notification deleted successfully",
	})
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, statusCode int, response models.APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
