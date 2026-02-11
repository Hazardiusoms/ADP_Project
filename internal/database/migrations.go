package database

import (
	"log"
)

// MigrateDatabase применяет миграции базы данных
func MigrateDatabase() {
	mu.Lock()
	defer mu.Unlock()

	// добавляем колонку images в bookings если ее нет
	addImagesToBookings := `
		ALTER TABLE bookings 
		ADD COLUMN images TEXT;
	`
	
	// sqlite не поддерживает IF NOT EXISTS для ALTER TABLE ADD COLUMN
	// поэтому используем обходной путь - пробуем добавить, игнорируем ошибку если уже есть
	_, err := DB.Exec(addImagesToBookings)
	if err != nil {
		// колонка может уже существовать, это нормально
		log.Printf("Note: images column for bookings (may already exist): %v", err)
	} else {
		log.Println("Added images column to bookings table")
	}

	// добавляем колонку images в resources если ее нет
	addImagesToResources := `
		ALTER TABLE resources 
		ADD COLUMN images TEXT;
	`
	
	_, err = DB.Exec(addImagesToResources)
	if err != nil {
		log.Printf("Note: images column for resources (may already exist): %v", err)
	} else {
		log.Println("Added images column to resources table")
	}

	// добавляем колонку payment_details в users если ее нет
	addPaymentDetailsToUsers := `
		ALTER TABLE users 
		ADD COLUMN payment_details TEXT;
	`
	
	_, err = DB.Exec(addPaymentDetailsToUsers)
	if err != nil {
		log.Printf("Note: payment_details column for users (may already exist): %v", err)
	} else {
		log.Println("Added payment_details column to users table")
	}

	// добавляем колонку location в resources если ее нет
	addLocationToResources := `
		ALTER TABLE resources 
		ADD COLUMN location TEXT DEFAULT 'Astana';
	`
	
	_, err = DB.Exec(addLocationToResources)
	if err != nil {
		log.Printf("Note: location column for resources (may already exist): %v", err)
	} else {
		log.Println("Added location column to resources table")
		// устанавливаем дефолтную локацию для существующих ресурсов
		DB.Exec("UPDATE resources SET location = 'Astana' WHERE location IS NULL")
	}

	// добавляем колонку booking_type в resources если ее нет
	addBookingTypeToResources := `
		ALTER TABLE resources 
		ADD COLUMN booking_type TEXT DEFAULT 'booking';
	`
	
	_, err = DB.Exec(addBookingTypeToResources)
	if err != nil {
		log.Printf("Note: booking_type column for resources (may already exist): %v", err)
	} else {
		log.Println("Added booking_type column to resources table")
		// устанавливаем дефолтный тип для существующих ресурсов
		DB.Exec("UPDATE resources SET booking_type = 'booking' WHERE booking_type IS NULL")
	}

	// добавляем колонку sale_price в resources если ее нет
	addSalePriceToResources := `
		ALTER TABLE resources 
		ADD COLUMN sale_price REAL DEFAULT 0;
	`
	
	_, err = DB.Exec(addSalePriceToResources)
	if err != nil {
		log.Printf("Note: sale_price column for resources (may already exist): %v", err)
	} else {
		log.Println("Added sale_price column to resources table")
	}
}
