package database

import (
	"database/sql"
	"log"
	"sync"

	_ "modernc.org/sqlite"
)

var DB *sql.DB
var mu sync.RWMutex // мьютекс для безопасного доступа к бд

func InitDB() {
	var err error
	// открываем sqlite с настройками для конкурентного доступа
	DB, err = sql.Open("sqlite", "./gobook.db?_busy_timeout=1000&_journal_mode=WAL&_synchronous=NORMAL")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	// настройки пула соединений для лучшей конкурентности
	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(0)

	// включаем внешние ключи
	DB.Exec("PRAGMA foreign_keys = ON")
	// таймаут ожидания блокировки 1 секунда
	DB.Exec("PRAGMA busy_timeout = 1000")
	// включаем WAL режим для конкурентных чтений
	DB.Exec("PRAGMA journal_mode = WAL")
	// синхронизация NORMAL для лучшей производительности
	DB.Exec("PRAGMA synchronous = NORMAL")

	createTables()
	MigrateDatabase() // применяем миграции
	log.Println("Database initialized successfully")
}

func createTables() {
	usersTable := `CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		full_name TEXT NOT NULL,
		phone TEXT,
		role TEXT DEFAULT "user",
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		is_active BOOLEAN DEFAULT 1
	);`

	resourcesTable := `CREATE TABLE IF NOT EXISTS resources (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		category TEXT NOT NULL,
		base_price REAL NOT NULL DEFAULT 0,
		pricing_rules TEXT,
		max_duration_hours INTEGER DEFAULT 24,
		min_booking_hours INTEGER DEFAULT 1,
		availability_schedule TEXT,
		created_by INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		is_active BOOLEAN DEFAULT 1,
		FOREIGN KEY(created_by) REFERENCES users(id)
	);`

	bookingsTable := `CREATE TABLE IF NOT EXISTS bookings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		resource_id INTEGER NOT NULL,
		start_time DATETIME NOT NULL,
		end_time DATETIME NOT NULL,
		status TEXT DEFAULT "pending",
		total_price REAL DEFAULT 0,
		metadata TEXT,
		created_by INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		cancellation_reason TEXT,
		FOREIGN KEY(user_id) REFERENCES users(id),
		FOREIGN KEY(resource_id) REFERENCES resources(id),
		FOREIGN KEY(created_by) REFERENCES users(id)
	);`

	paymentsTable := `CREATE TABLE IF NOT EXISTS payments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		booking_id INTEGER NOT NULL,
		payment_method TEXT NOT NULL,
		amount REAL NOT NULL,
		currency TEXT DEFAULT "USD",
		status TEXT DEFAULT "pending",
		transaction_id TEXT,
		payment_details TEXT,
		payment_date DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(booking_id) REFERENCES bookings(id)
	);`

	notificationsTable := `CREATE TABLE IF NOT EXISTS notifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		type TEXT NOT NULL,
		subject TEXT,
		content TEXT NOT NULL,
		status TEXT DEFAULT "pending",
		sent_at DATETIME,
		read_at DATETIME,
		metadata TEXT,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);`

	auditLogsTable := `CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		action TEXT NOT NULL,
		user_id INTEGER,
		entity_type TEXT NOT NULL,
		entity_id INTEGER NOT NULL,
		old_values TEXT,
		new_values TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		ip_address TEXT
	);`

	reviewsTable := `CREATE TABLE IF NOT EXISTS reviews (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		resource_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		rating INTEGER DEFAULT 5,
		comment TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(resource_id) REFERENCES resources(id),
		FOREIGN KEY(user_id) REFERENCES users(id)
	);`

	mu.Lock()
	defer mu.Unlock()

	DB.Exec(usersTable)
	DB.Exec(resourcesTable)
	DB.Exec(bookingsTable)
	DB.Exec(paymentsTable)
	DB.Exec(notificationsTable)
	DB.Exec(auditLogsTable)
	DB.Exec(reviewsTable)

	log.Println("All tables created successfully")
}

// SafeQuery выполняет запрос - блокировка не нужна с WAL режимом для чтений
// WAL режим позволяет конкурентные чтения, поэтому блокировки чтения не нужны
func SafeQuery(query string, args ...interface{}) (*sql.Rows, error) {
	return DB.Query(query, args...)
}

// SafeQueryRow выполняет запрос одной строки - блокировка не нужна с WAL режимом для чтений
func SafeQueryRow(query string, args ...interface{}) *sql.Row {
	return DB.QueryRow(query, args...)
}

// SafeExec выполняет операцию записи с блокировкой записи
// записи все еще нуждаются в защите чтобы избежать конфликтов
func SafeExec(query string, args ...interface{}) (sql.Result, error) {
	mu.Lock()
	defer mu.Unlock()
	return DB.Exec(query, args...)
}
