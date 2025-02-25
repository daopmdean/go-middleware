package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// In-memory rate limit storage (for demo purposes)
var rateLimitStore = make(map[string]time.Time)
var mu sync.Mutex

// LoggingMiddleware logs request details
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r) // Call the next middleware/handler
		duration := time.Since(start)
		log.Printf("Completed in %v", duration)
	})
}

// AuthMiddleware checks for an API key in headers
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		fmt.Println("X-API-Key", apiKey)
		apikey := r.Header.Get("x-API-key")
		fmt.Println("x-API-key", apikey)
		if apiKey != "secret123" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r) // Call the next middleware/handler
	})
}

// RateLimitMiddleware limits requests per IP (1 request per 5 seconds)
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getRealIp(r)
		fmt.Println(ip)

		mu.Lock()
		lastRequest, found := rateLimitStore[ip]
		mu.Unlock()

		if found && time.Since(lastRequest) < 5*time.Second {
			http.Error(w, "Too many requests. Please wait.", http.StatusTooManyRequests)
			return
		}

		// Update last request time
		mu.Lock()
		rateLimitStore[ip] = time.Now()
		mu.Unlock()

		next.ServeHTTP(w, r) // Call the next middleware/handler
	})
}

func getRealIp(r *http.Request) string {
	if ip := r.Header.Get("X-Real-Ip"); ip != "" {
		return ip
	}

	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		// X-Forwarded-For is potentially a list of addresses separated with ","
		parts := strings.Split(ip, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				return p
			}
		}
	}

	return strings.Split(r.RemoteAddr, ":")[0]
}

// HelloHandler is the final HTTP handler
type HelloHandler struct{}

func (h HelloHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello from HelloHandler!")
}

func main() {
	helloHandler := HelloHandler{}                                                         // Base handler
	wrappedHandler := LoggingMiddleware(AuthMiddleware(RateLimitMiddleware(helloHandler))) // Chain middleware

	http.Handle("/hello", wrappedHandler) // Register the wrapped handler
	log.Println("Server starting on :8080...")
	http.ListenAndServe(":8080", nil)
}
