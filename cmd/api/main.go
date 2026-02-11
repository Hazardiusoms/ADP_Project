package main

import (
	"log"
	"net/http"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/auth"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/booking"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/resource"
)

func main() {
	database.InitDB()

	// Static Files
	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// --- Auth Module Routes ---
	http.HandleFunc("/login", auth.LoginHandler)
	http.HandleFunc("/register", auth.RegisterHandler)

	// --- Resource Module Routes ---
	// Middleware should go here, simplified for brevity
	http.HandleFunc("/dashboard", resource.DashboardHandler)
	http.HandleFunc("/resources/create", resource.CreateResourceHandler)

	// --- Booking Module Routes ---
	http.HandleFunc("/bookings/create", booking.CreateBookingHandler)

	log.Println("GoBook Monolith starting on :8080...")
	http.ListenAndServe(":8080", nil)
}
