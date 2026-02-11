package middleware

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Hazardiusoms/ADP_Project/internal/modules/auth"
)

// Middleware тип для цепочки middleware функций
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Chain объединяет несколько middleware функций в цепочку
func Chain(middlewares ...Middleware) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		// проходим по middleware в обратном порядке
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// LoggingMiddleware логирует http запросы с методом, путем и длительностью
func LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// засекаем время начала
		start := time.Now()
		
		// создаем обертку для response writer чтобы захватить статус код
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		// вызываем следующий обработчик
		next(rw, r)
		
		// логируем запрос
		duration := time.Since(start)
		log.Printf("[%s] %s %s - %d - %v",
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
			rw.statusCode,
			duration,
		)
	}
}

// responseWriter обертка для http.ResponseWriter чтобы захватить статус код
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	// сохраняем статус код
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// AuthMiddleware проверяет что пользователь авторизован перед доступом к защищенным маршрутам
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// получаем текущего пользователя
		user, err := auth.GetCurrentUser(r)
		if err != nil {
			// проверяем если это json запрос
			if r.Header.Get("Content-Type") == "application/json" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"success":false,"error":"Unauthorized"}`))
				return
			}
			// редирект на логин для html запросов
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		
		// сохраняем данные пользователя в заголовках запроса (можно получить в обработчиках)
		// для простоты передаем через кастомные заголовки
		r.Header.Set("X-User-ID", strconv.Itoa(user.ID))
		r.Header.Set("X-User-Email", user.Email)
		r.Header.Set("X-User-Role", user.Role)
		
		next(w, r)
	}
}

// AdminMiddleware проверяет что у пользователя роль админа
func AdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// получаем текущего пользователя
		user, err := auth.GetCurrentUser(r)
		if err != nil {
			// если json запрос - возвращаем json ошибку
			if r.Header.Get("Content-Type") == "application/json" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"success":false,"error":"Unauthorized"}`))
				return
			}
			// иначе редирект на логин
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		
		if user.Role != "admin" {
			if r.Header.Get("Content-Type") == "application/json" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"success":false,"error":"Forbidden: Admin access required"}`))
				return
			}
			http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
			return
		}
		
		next(w, r)
	}
}

// CORSMiddleware adds CORS headers to responses
func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next(w, r)
	}
}

// JSONMiddleware sets Content-Type to application/json
func JSONMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next(w, r)
	}
}

// RateLimitMiddleware is a simple rate limiter (can be enhanced)
func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Simple rate limiting - in production, use a proper rate limiter
		// This is a placeholder that can be enhanced
		next(w, r)
	}
}

// ApplyMiddleware applies middleware to a handler function
func ApplyMiddleware(handler http.HandlerFunc, middlewares ...Middleware) http.HandlerFunc {
	return Chain(middlewares...)(handler)
}
