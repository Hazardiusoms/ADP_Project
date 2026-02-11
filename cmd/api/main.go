package main

import (
	"log"
	"net/http"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
	"github.com/Hazardiusoms/ADP_Project/internal/middleware"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/auth"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/booking"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/notification"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/payment"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/resource"
	"github.com/Hazardiusoms/ADP_Project/internal/services"
)

func main() {
	// инициализируем бд
	database.InitDB()

	// создаем админа по умолчанию если его нет
	database.CreateDefaultAdmin()

	// запускаем фоновые воркеры (горутины для асинхронной обработки)
	worker := services.NewBackgroundWorker()
	worker.Start()
	defer worker.Stop()

	// запускаем асинхронный обработчик уведомлений
	services.SendNotificationAsync(services.NotificationChan)

	// раздаем статические файлы (css/js) и загрузки
	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	uploads := http.FileServer(http.Dir("./web/uploads"))
	http.Handle("/uploads/", http.StripPrefix("/uploads/", uploads))

	// определяем маршруты с middleware

	// публичные маршруты (без авторизации)
	http.HandleFunc("/", middleware.ApplyMiddleware(
		func(w http.ResponseWriter, r *http.Request) {
			// редирект на логин если корень
			if r.URL.Path == "/" {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
			}
		},
		middleware.LoggingMiddleware,
	))

	// маршруты авторизации (публичные, но с логированием)
	http.HandleFunc("/login", middleware.ApplyMiddleware(
		auth.LoginHandler,
		middleware.LoggingMiddleware,
	))
	http.HandleFunc("/register", middleware.ApplyMiddleware(
		auth.RegisterHandler,
		middleware.LoggingMiddleware,
	))
	// профиль требует авторизации
	http.HandleFunc("/profile", middleware.ApplyMiddleware(
		auth.ProfileHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
	))
	http.HandleFunc("/profile/update", middleware.ApplyMiddleware(
		auth.UpdateProfileHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
	))

	// json api для авторизации (публичные)
	http.HandleFunc("/api/auth/register", middleware.ApplyMiddleware(
		auth.RegisterHandler,
		middleware.LoggingMiddleware,
		middleware.CORSMiddleware,
	))
	http.HandleFunc("/api/auth/login", middleware.ApplyMiddleware(
		auth.LoginHandler,
		middleware.LoggingMiddleware,
		middleware.CORSMiddleware,
	))

	// просмотр бронирований ресурса (для создателя ресурса)
	http.HandleFunc("/resources/bookings", middleware.ApplyMiddleware(
		resource.ViewResourceBookingsHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
	))

	// админские маршруты (для управления пользователями)
	http.HandleFunc("/admin/users", middleware.ApplyMiddleware(
		auth.AdminUsersHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.AdminMiddleware,
	))
	http.HandleFunc("/admin/users/delete", middleware.ApplyMiddleware(
		auth.DeleteUserHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.AdminMiddleware,
	))
	http.HandleFunc("/admin/users/reset-password", middleware.ApplyMiddleware(
		auth.ResetPasswordHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.AdminMiddleware,
	))
	http.HandleFunc("/admin/users/edit", middleware.ApplyMiddleware(
		auth.EditUserHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.AdminMiddleware,
	))
	http.HandleFunc("/admin/users/update", middleware.ApplyMiddleware(
		auth.UpdateUserHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.AdminMiddleware,
	))
	http.HandleFunc("/admin/audit", middleware.ApplyMiddleware(
		auth.AdminAuditLogHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.AdminMiddleware,
	))
	http.HandleFunc("/admin/delete-resource", middleware.ApplyMiddleware(
		auth.DeleteResourceAdminHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.AdminMiddleware,
	))
	http.HandleFunc("/admin/delete-booking", middleware.ApplyMiddleware(
		auth.DeleteBookingAdminHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.AdminMiddleware,
	))

	// защищенные маршруты (требуют авторизации)
	// модуль ресурсов - эндпоинты ленары (crud)
	http.HandleFunc("/dashboard", middleware.ApplyMiddleware(
		resource.DashboardHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
	))
	// просмотр ресурса (публичный, но лучше авторизоваться)
	http.HandleFunc("/resources/view", middleware.ApplyMiddleware(
		resource.ViewResourceHandler,
		middleware.LoggingMiddleware,
	))
	// создание ресурса
	http.HandleFunc("/resources/create", middleware.ApplyMiddleware(
		resource.CreateResourceHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
	))
	// редактирование ресурса
	http.HandleFunc("/resources/edit", middleware.ApplyMiddleware(
		resource.EditResourceHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
	))
	// обновление ресурса
	http.HandleFunc("/resources/update", middleware.ApplyMiddleware(
		resource.UpdateResourceFormHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
	))
	// удаление изображения ресурса
	http.HandleFunc("/resources/delete-image", middleware.ApplyMiddleware(
		resource.DeleteResourceImageHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
	))
	// создание отзыва на ресурс
	http.HandleFunc("/resources/review", middleware.ApplyMiddleware(
		resource.CreateReviewHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
	))

	// json api для ресурсов - crud ленары (4+ эндпоинта)
	http.HandleFunc("/api/resources", middleware.ApplyMiddleware(
		func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "GET":
				resource.GetResourcesHandler(w, r)
			case "POST":
				resource.CreateResourceHandler(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		},
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.CORSMiddleware,
	))
	http.HandleFunc("/api/resources/get", middleware.ApplyMiddleware(
		resource.GetResourceHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.CORSMiddleware,
	))
	http.HandleFunc("/api/resources/update", middleware.ApplyMiddleware(
		resource.UpdateResourceHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.CORSMiddleware,
	))
	http.HandleFunc("/api/resources/delete", middleware.ApplyMiddleware(
		resource.DeleteResourceHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.CORSMiddleware,
	))

	// модуль уведомлений - дополнительный crud ленары (5 эндпоинтов)
	http.HandleFunc("/api/notifications", middleware.ApplyMiddleware(
		notification.GetNotificationsHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.CORSMiddleware,
	))
	http.HandleFunc("/api/notifications/get", middleware.ApplyMiddleware(
		notification.GetNotificationHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.CORSMiddleware,
	))
	// создание уведомлений только для админа
	http.HandleFunc("/api/notifications/create", middleware.ApplyMiddleware(
		notification.CreateNotificationHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.AdminMiddleware,
		middleware.CORSMiddleware,
	))
	http.HandleFunc("/api/notifications/update", middleware.ApplyMiddleware(
		notification.UpdateNotificationHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.CORSMiddleware,
	))
	http.HandleFunc("/api/notifications/delete", middleware.ApplyMiddleware(
		notification.DeleteNotificationHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.CORSMiddleware,
	))

	// модуль бронирований - эндпоинты нурсалима (3+ эндпоинта)
	http.HandleFunc("/bookings/create", middleware.ApplyMiddleware(
		booking.CreateBookingHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
	))
	http.HandleFunc("/bookings/my", middleware.ApplyMiddleware(
		booking.ListBookingsHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
	))
	http.HandleFunc("/bookings/edit", middleware.ApplyMiddleware(
		booking.EditBookingHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
	))
	http.HandleFunc("/bookings/update", middleware.ApplyMiddleware(
		booking.UpdateBookingHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
	))
	http.HandleFunc("/bookings/confirm", middleware.ApplyMiddleware(
		booking.ConfirmBookingHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
	))
	http.HandleFunc("/bookings/upload-image", middleware.ApplyMiddleware(
		booking.UploadImageHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
	))
	http.HandleFunc("/bookings/delete-image", middleware.ApplyMiddleware(
		booking.DeleteImageHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
	))

	// json api для бронирований - эндпоинты нурсалима
	http.HandleFunc("/api/bookings", middleware.ApplyMiddleware(
		func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "GET":
				booking.GetBookingsHandler(w, r)
			case "POST":
				booking.CreateBookingHandler(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		},
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.CORSMiddleware,
	))
	http.HandleFunc("/api/bookings/cancel", middleware.ApplyMiddleware(
		booking.CancelBookingHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.CORSMiddleware,
	))

	// модуль платежей - crud нурсалима (4 эндпоинта)
	http.HandleFunc("/api/payments", middleware.ApplyMiddleware(
		payment.GetPaymentsHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.CORSMiddleware,
	))
	http.HandleFunc("/api/payments/get", middleware.ApplyMiddleware(
		payment.GetPaymentHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.CORSMiddleware,
	))
	// обновление и удаление платежей только для админа
	http.HandleFunc("/api/payments/update", middleware.ApplyMiddleware(
		payment.UpdatePaymentHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.AdminMiddleware,
		middleware.CORSMiddleware,
	))
	http.HandleFunc("/api/payments/delete", middleware.ApplyMiddleware(
		payment.DeletePaymentHandler,
		middleware.LoggingMiddleware,
		middleware.AuthMiddleware,
		middleware.AdminMiddleware,
		middleware.CORSMiddleware,
	))

	// запускаем сервер на порту 8081
	log.Println("Server is launched on localhost:8081")
	log.Println("------------------------------------------------")

	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}
