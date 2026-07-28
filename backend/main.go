package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed frontend
var frontendFiles embed.FS

func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func main() {
	connectDB()

	mux := http.NewServeMux()

	// Public
	mux.HandleFunc("/api/signup", withCORS(handleSignup))
	mux.HandleFunc("/api/login", withCORS(handleLogin))
	mux.HandleFunc("/api/admin/login", withCORS(handleAdminLogin))
	mux.HandleFunc("/api/products", withCORS(handleListProducts))
	mux.HandleFunc("/api/products/", withCORS(handleGetProduct))
	mux.HandleFunc("/api/paystack/webhook", withCORS(handlePaystackWebhook))

	// Customer-only
	mux.HandleFunc("/api/orders", withCORS(requireAuth("customer", handleCreateOrder)))
	mux.HandleFunc("/api/orders/history", withCORS(requireAuth("customer", handleOrderHistory)))

	// Admin-only
	mux.HandleFunc("/api/admin/orders", withCORS(requireAuth("admin", handleAdminListOrders)))
	mux.HandleFunc("/api/admin/orders/", withCORS(requireAuth("admin", handleAdminOrderByID)))
	mux.HandleFunc("/api/admin/products", withCORS(requireAuth("admin", handleAdminCreateProduct)))
	mux.HandleFunc("/api/admin/products/", withCORS(requireAuth("admin", handleAdminProductByID)))
	mux.HandleFunc("/api/admin/stats", withCORS(requireAuth("admin", handleAdminStats)))

	// Serve the PWA frontend from the files embedded in this binary
	frontendRoot, err := fs.Sub(frontendFiles, "frontend")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServer(http.FS(frontendRoot)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Bloom Scents server running on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
