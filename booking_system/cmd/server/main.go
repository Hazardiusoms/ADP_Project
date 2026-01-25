package main

import (
	"ADP_Project/booking_system/internal/handlers"
	"fmt"
	"net/http"
)

func main() {

	http.HandleFunc("/", handlers.HomeHandler)       // Было HomeHandler
	http.HandleFunc("/rooms", handlers.GetRooms)     // Было GetRooms
	http.HandleFunc("/book", handlers.CreateBooking) // Было CreateBooking

	// server launch
	port := ":8080"
	fmt.Printf("Starting Booking System on port %s...\n", port)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
