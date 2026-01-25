package handlers

import (
	"fmt"
	"net/http"
)

// HomeHandler - just check
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Booking System API v1.0 is RUNNING")
}

// GetRooms - must return list of rooms
func GetRooms(w http.ResponseWriter, r *http.Request) {
	// in future there will be DB request
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`[{"id": 1, "name": "Conference Room A", "capacity": 10}]`))
}

// CreateBooking - creating booking(right now just blocks)
func CreateBooking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status": "booking created"}`))
}
