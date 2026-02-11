package payment

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
	"github.com/Hazardiusoms/ADP_Project/internal/models"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/auth"
)

// GetPaymentsHandler returns all payments for the current user (JSON API)
func GetPaymentsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetCurrentUser(r)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	// Get payments for user's bookings
	rows, err := database.SafeQuery(
		`SELECT p.id, p.booking_id, p.payment_method, p.amount, p.currency, 
		 p.status, p.transaction_id, p.payment_date, p.created_at
		 FROM payments p
		 JOIN bookings b ON p.booking_id = b.id
		 WHERE b.user_id = ? 
		 ORDER BY p.created_at DESC`,
		user.ID,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Error fetching payments",
		})
		return
	}
	defer rows.Close()

	var payments []models.Payment
	for rows.Next() {
		var payment models.Payment
		err := rows.Scan(
			&payment.ID, &payment.BookingID, &payment.PaymentMethod,
			&payment.Amount, &payment.Currency, &payment.Status,
			&payment.TransactionID, &payment.PaymentDate, &payment.CreatedAt,
		)
		if err != nil {
			continue
		}
		payments = append(payments, payment)
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    payments,
	})
}

// GetPaymentHandler returns a single payment by ID (JSON API)
func GetPaymentHandler(w http.ResponseWriter, r *http.Request) {
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
			Error:   "Invalid payment ID",
		})
		return
	}

	var payment models.Payment
	var bookingUserID int
	err = database.SafeQueryRow(
		`SELECT p.id, p.booking_id, p.payment_method, p.amount, p.currency, 
		 p.status, p.transaction_id, p.payment_date, p.created_at, b.user_id
		 FROM payments p
		 JOIN bookings b ON p.booking_id = b.id
		 WHERE p.id = ?`,
		id,
	).Scan(
		&payment.ID, &payment.BookingID, &payment.PaymentMethod,
		&payment.Amount, &payment.Currency, &payment.Status,
		&payment.TransactionID, &payment.PaymentDate, &payment.CreatedAt,
		&bookingUserID,
	)

	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "Payment not found",
		})
		return
	}

	// Verify ownership (user can only see their own payments, admin can see all)
	if bookingUserID != user.ID && user.Role != "admin" {
		respondJSON(w, http.StatusForbidden, models.APIResponse{
			Success: false,
			Error:   "Not authorized to view this payment",
		})
		return
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    payment,
	})
}

// UpdatePaymentHandler updates payment status (JSON API)
func UpdatePaymentHandler(w http.ResponseWriter, r *http.Request) {
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
			Error:   "Invalid payment ID",
		})
		return
	}

	var req struct {
		Status        string  `json:"status"`
		TransactionID string  `json:"transaction_id"`
		Amount        float64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	// Only admin can update payments
	if user.Role != "admin" {
		respondJSON(w, http.StatusForbidden, models.APIResponse{
			Success: false,
			Error:   "Forbidden: Admin access required",
		})
		return
	}

	// Update payment
	updateQuery := "UPDATE payments SET status = ?, transaction_id = ?"
	args := []interface{}{req.Status, req.TransactionID}
	
	if req.Amount > 0 {
		updateQuery += ", amount = ?"
		args = append(args, req.Amount)
	}
	
	updateQuery += " WHERE id = ?"
	args = append(args, id)

	_, err = database.SafeExec(updateQuery, args...)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Error updating payment",
		})
		return
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Payment updated successfully",
	})
}

// DeletePaymentHandler deletes a payment (soft delete - JSON API)
func DeletePaymentHandler(w http.ResponseWriter, r *http.Request) {
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

	// Only admin can delete payments
	if user.Role != "admin" {
		respondJSON(w, http.StatusForbidden, models.APIResponse{
			Success: false,
			Error:   "Forbidden: Admin access required",
		})
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid payment ID",
		})
		return
	}

	// In a real system, we might soft delete or archive
	// For now, we'll just mark as cancelled
	_, err = database.SafeExec(
		"UPDATE payments SET status = 'cancelled' WHERE id = ?",
		id,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Error deleting payment",
		})
		return
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Payment deleted successfully",
	})
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, statusCode int, response models.APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
