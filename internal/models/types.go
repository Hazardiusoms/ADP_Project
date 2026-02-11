package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// User matches ERD: USERS table
type User struct {
	ID           int       `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	FullName     string    `json:"full_name" db:"full_name"`
	Phone        string    `json:"phone" db:"phone"`
	Role         string    `json:"role" db:"role"` // "admin", "user"
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	IsActive     bool      `json:"is_active" db:"is_active"`
}

// Resource matches ERD: RESOURCES table
type Resource struct {
	ID                int             `json:"id" db:"id"`
	Name              string          `json:"name" db:"name"`
	Description       string          `json:"description" db:"description"`
	Category          string          `json:"category" db:"category"`
	Location          string          `json:"location" db:"location"`
	BasePrice         float64         `json:"base_price" db:"base_price"`
	SalePrice         float64         `json:"sale_price" db:"sale_price"`
	BookingType       string          `json:"booking_type" db:"booking_type"` // "booking" or "sale"
	PricingRules      JSONMap         `json:"pricing_rules" db:"pricing_rules"`
	MaxDurationHours  int             `json:"max_duration_hours" db:"max_duration_hours"`
	MinBookingHours   int             `json:"min_booking_hours" db:"min_booking_hours"`
	AvailabilitySchedule JSONMap      `json:"availability_schedule" db:"availability_schedule"`
	CreatedBy         int             `json:"created_by" db:"created_by"`
	CreatedAt         time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at" db:"updated_at"`
	IsActive          bool            `json:"is_active" db:"is_active"`
}

// Booking matches ERD: BOOKINGS table
type Booking struct {
	ID               int       `json:"id" db:"id"`
	UserID           int       `json:"user_id" db:"user_id"`
	ResourceID       int       `json:"resource_id" db:"resource_id"`
	StartTime        time.Time `json:"start_time" db:"start_time"`
	EndTime          time.Time `json:"end_time" db:"end_time"`
	Status           string    `json:"status" db:"status"` // "pending", "confirmed", "completed", "cancelled"
	TotalPrice       float64   `json:"total_price" db:"total_price"`
	Metadata         JSONMap   `json:"metadata" db:"metadata"`
	Images           string    `json:"images,omitempty" db:"images"` // JSON array of image paths
	CreatedBy        int       `json:"created_by" db:"created_by"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
	CancellationReason string  `json:"cancellation_reason,omitempty" db:"cancellation_reason"`
}

// Payment matches ERD: PAYMENTS table
type Payment struct {
	ID            int       `json:"id" db:"id"`
	BookingID     int       `json:"booking_id" db:"booking_id"`
	PaymentMethod string    `json:"payment_method" db:"payment_method"`
	Amount        float64   `json:"amount" db:"amount"`
	Currency      string    `json:"currency" db:"currency"`
	Status        string    `json:"status" db:"status"` // "pending", "completed", "failed", "refunded"
	TransactionID string    `json:"transaction_id" db:"transaction_id"`
	PaymentDetails JSONMap  `json:"payment_details" db:"payment_details"`
	PaymentDate   time.Time `json:"payment_date" db:"payment_date"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// Notification matches ERD: NOTIFICATIONS table
type Notification struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Type      string    `json:"type" db:"type"` // "booking_confirmation", "reminder", "cancellation"
	Subject   string    `json:"subject" db:"subject"`
	Content   string    `json:"content" db:"content"`
	Status    string    `json:"status" db:"status"` // "sent", "pending", "failed"
	SentAt    *time.Time `json:"sent_at,omitempty" db:"sent_at"`
	ReadAt    *time.Time `json:"read_at,omitempty" db:"read_at"`
	Metadata  JSONMap   `json:"metadata" db:"metadata"`
}

// AuditLog matches ERD: AUDIT_LOGS table
type AuditLog struct {
	ID         int       `json:"id" db:"id"`
	Action     string    `json:"action" db:"action"`
	UserID     *int      `json:"user_id,omitempty" db:"user_id"`
	EntityType string    `json:"entity_type" db:"entity_type"`
	EntityID   int       `json:"entity_id" db:"entity_id"`
	OldValues  JSONMap   `json:"old_values" db:"old_values"`
	NewValues  JSONMap   `json:"new_values" db:"new_values"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	IPAddress  string    `json:"ip_address" db:"ip_address"`
}

// JSONMap is a helper type for JSON fields in database
type JSONMap map[string]interface{}

// Value implements driver.Valuer interface
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		*j = make(JSONMap)
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Request/Response DTOs for API

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateResourceRequest struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Category          string   `json:"category"`
	BasePrice         float64  `json:"base_price"`
	MaxDurationHours  int      `json:"max_duration_hours"`
	MinBookingHours   int      `json:"min_booking_hours"`
}

type CreateBookingRequest struct {
	ResourceID int       `json:"resource_id"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
