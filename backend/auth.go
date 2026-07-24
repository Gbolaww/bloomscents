package main

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte(getEnvOrDefault("JWT_SECRET", "change-this-secret-in-production"))

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type signupRequest struct {
	FullName        string `json:"full_name"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	Password        string `json:"password"`
	DeliveryAddress string `json:"delivery_address"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// POST /api/signup
func handleSignup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" || req.FullName == "" {
		http.Error(w, "full_name, email and password are required", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "failed to process password", http.StatusInternalServerError)
		return
	}

	var id int
	err = db.QueryRow(
		`INSERT INTO customers (full_name, email, phone, password_hash, delivery_address)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		req.FullName, req.Email, req.Phone, string(hash), req.DeliveryAddress,
	).Scan(&id)
	if err != nil {
		http.Error(w, "email already in use or invalid data", http.StatusConflict)
		return
	}

	token, err := makeToken(id, "customer")
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"token": token, "customer_id": id})
}

// POST /api/login
func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var id int
	var hash string
	err := db.QueryRow(`SELECT id, password_hash FROM customers WHERE email = $1`, req.Email).Scan(&id, &hash)
	if err != nil {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	token, err := makeToken(id, "customer")
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"token": token, "customer_id": id})
}

// POST /api/admin/login
func handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var id int
	var hash string
	err := db.QueryRow(`SELECT id, password_hash FROM admins WHERE email = $1`, req.Email).Scan(&id, &hash)
	if err != nil {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	token, err := makeToken(id, "admin")
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"token": token, "admin_id": id})
}

func makeToken(subjectID int, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  subjectID,
		"role": role,
		"exp":  time.Now().Add(30 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// requireAuth wraps a handler, checking for a valid bearer token of the given role.
// role should be "customer" or "admin".
func requireAuth(role string, next func(w http.ResponseWriter, r *http.Request, subjectID int)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			http.Error(w, "missing or invalid authorization header", http.StatusUnauthorized)
			return
		}
		tokenStr := authHeader[7:]

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "invalid token claims", http.StatusUnauthorized)
			return
		}

		if claims["role"] != role {
			http.Error(w, "insufficient permissions", http.StatusForbidden)
			return
		}

		subFloat, ok := claims["sub"].(float64)
		if !ok {
			http.Error(w, "invalid token subject", http.StatusUnauthorized)
			return
		}

		next(w, r, int(subFloat))
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
