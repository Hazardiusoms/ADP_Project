package models

import "time"

// User
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"` // "admin" or "client"
}

// Room
type Room struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
	Location string `json:"location"`
}

// Booking
type Booking struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	RoomID    int       `json:"room_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"` // "confirmed", "cancelled"
}
