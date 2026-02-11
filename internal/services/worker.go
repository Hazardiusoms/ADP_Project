package services

import (
	"log"
	"time"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
)

// BackgroundWorker обрабатывает асинхронные задачи
type BackgroundWorker struct {
	stopChan chan bool
	running  bool
}

// NewBackgroundWorker создает новый фоновый воркер
func NewBackgroundWorker() *BackgroundWorker {
	return &BackgroundWorker{
		stopChan: make(chan bool),
		running:  false,
	}
}

// Start запускает фоновый воркер
func (w *BackgroundWorker) Start() {
	// если уже запущен - выходим
	if w.running {
		return
	}
	w.running = true
	log.Println("Background worker started")

	// запускаем горутины для разных задач
	go w.processNotifications()
	go w.cleanupExpiredBookings()
	go w.processPendingPayments()
}

// Stop останавливает фоновый воркер
func (w *BackgroundWorker) Stop() {
	// если не запущен - выходим
	if !w.running {
		return
	}
	w.running = false
	close(w.stopChan)
	log.Println("Background worker stopped")
}

// processNotifications обрабатывает ожидающие уведомления
func (w *BackgroundWorker) processNotifications() {
	// тикер каждые 30 секунд
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			// если получили сигнал остановки - выходим
			return
		case <-ticker.C:
			// каждые 30 секунд отправляем уведомления
			w.sendPendingNotifications()
		}
	}
}

// sendPendingNotifications отправляет все ожидающие уведомления
func (w *BackgroundWorker) sendPendingNotifications() {
	// получаем список ожидающих уведомлений (максимум 10)
	rows, err := database.SafeQuery(
		"SELECT id, user_id, type, subject, content FROM notifications WHERE status = 'pending' LIMIT 10",
	)
	if err != nil {
		log.Printf("Error fetching pending notifications: %v", err)
		return
	}
	defer rows.Close()

	// обрабатываем каждое уведомление
	for rows.Next() {
		var id, userID int
		var notifType, subject, content string
		if err := rows.Scan(&id, &userID, &notifType, &subject, &content); err != nil {
			continue
		}

		// симулируем отправку уведомления (в реальном приложении отправляли бы email/sms)
		log.Printf("[Worker] Sending notification #%d to user #%d: %s", id, userID, subject)

		// обновляем статус на отправлено
		now := time.Now()
		_, err = database.SafeExec(
			"UPDATE notifications SET status = 'sent', sent_at = ? WHERE id = ?",
			now, id,
		)
		if err != nil {
			log.Printf("Error updating notification #%d: %v", id, err)
			// продолжаем со следующим уведомлением вместо ошибки
		}
	}
}

// cleanupExpiredBookings помечает истекшие бронирования как завершенные
func (w *BackgroundWorker) cleanupExpiredBookings() {
	ticker := time.NewTicker(5 * time.Minute) // уменьшенная частота чтобы избежать блокировок
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			now := time.Now()
			result, err := database.SafeExec(
				"UPDATE bookings SET status = 'completed' WHERE status = 'confirmed' AND end_time < ?",
				now,
			)
			if err != nil {
				// не логируем каждую ошибку busy, только если она персистирует
				if err.Error() != "database is locked" && err.Error() != "SQLITE_BUSY" {
					log.Printf("Error cleaning up bookings: %v", err)
				}
				continue
			}
			if rows, _ := result.RowsAffected(); rows > 0 {
				log.Printf("[Worker] Marked %d bookings as completed", rows)
			}
		}
	}
}

// processPendingPayments обрабатывает ожидающие платежи (симуляция)
func (w *BackgroundWorker) processPendingPayments() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			rows, err := database.SafeQuery(
				"SELECT id, booking_id, amount FROM payments WHERE status = 'pending' LIMIT 5",
			)
			if err != nil {
				log.Printf("Error fetching pending payments: %v", err)
				continue
			}
			defer rows.Close()

			for rows.Next() {
				var id, bookingID int
				var amount float64
				if err := rows.Scan(&id, &bookingID, &amount); err != nil {
					continue
				}

				// симулируем обработку платежа
				log.Printf("[Worker] Processing payment #%d for booking #%d: $%.2f", id, bookingID, amount)

				// помечаем как завершенный (симуляция)
				database.SafeExec(
					"UPDATE payments SET status = 'completed', payment_date = ? WHERE id = ?",
					time.Now(), id,
				)

				// обновляем статус бронирования
				database.SafeExec(
					"UPDATE bookings SET status = 'confirmed' WHERE id = ? AND status = 'pending'",
					bookingID,
				)
			}
		}
	}
}

// SendNotificationAsync отправляет уведомление асинхронно через канал
func SendNotificationAsync(notificationChan chan NotificationTask) {
	go func() {
		for task := range notificationChan {
			// вставляем уведомление в базу данных
			_, err := database.SafeExec(
				`INSERT INTO notifications (user_id, type, subject, content, status) 
				 VALUES (?, ?, ?, ?, 'pending')`,
				task.UserID, task.Type, task.Subject, task.Content,
			)
			if err != nil {
				log.Printf("Error creating notification: %v", err)
			} else {
				log.Printf("[Async] Notification queued for user #%d: %s", task.UserID, task.Subject)
			}
		}
	}()
}

// NotificationTask представляет уведомление которое нужно отправить
type NotificationTask struct {
	UserID  int
	Type    string
	Subject string
	Content string
}

// глобальный канал уведомлений
var NotificationChan = make(chan NotificationTask, 100)
