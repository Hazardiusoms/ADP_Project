package main

import (
	"booking-system/internal/handlers"
	"fmt"
	"net/http"
)

func main() {
	// routes
	http.HandleFunc("/", HomeHandler)
	http.HandleFunc("/rooms", GetRooms)
	http.HandleFunc("/book", CreateBooking)

	// server launch
	port := ":8080"
	fmt.Printf("Starting Booking System on port %s...\n", port)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
