package database

import (
	"log"

	"golang.org/x/crypto/bcrypt"
)

// CreateDefaultAdmin создает админа по умолчанию если его нет
func CreateDefaultAdmin() {
	// проверяем существует ли админ
	var existingEmail string
	err := SafeQueryRow("SELECT email FROM users WHERE email = ?", "admin@gobook.com").Scan(&existingEmail)
	if err == nil {
		// админ уже существует
		return
	}

	// админа нет, создаем его
	password := "admin123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Warning: Could not create default admin: %v", err)
		return
	}

	_, err = SafeExec(
		"INSERT INTO users (email, password_hash, full_name, phone, role) VALUES (?, ?, ?, ?, ?)",
		"admin@gobook.com", string(hashedPassword), "Administrator", "+77001234567", "admin",
	)
	if err != nil {
		log.Printf("Warning: Could not create default admin: %v", err)
		return
	}

	log.Println("========================================")
	log.Println("Default admin user created!")
	log.Println("Email: admin@gobook.com")
	log.Println("Password: admin123")
	log.Println("========================================")
}
