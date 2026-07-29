package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// googleTokenInfo mirrors the fields we need from Google's tokeninfo response.
type googleTokenInfo struct {
	Aud           string `json:"aud"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Name          string `json:"name"`
	Sub           string `json:"sub"`
}

// verifyGoogleIDToken confirms the token was issued by Google for our app (checking `aud`
// against GOOGLE_CLIENT_ID), then returns the verified email/name/subject.
func verifyGoogleIDToken(idToken string) (*googleTokenInfo, error) {
	expectedClientID := os.Getenv("GOOGLE_CLIENT_ID")
	if expectedClientID == "" {
		return nil, fmt.Errorf("GOOGLE_CLIENT_ID is not set on the server")
	}

	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google rejected the token")
	}

	var info googleTokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	if info.Aud != expectedClientID {
		return nil, fmt.Errorf("token was not issued for this app")
	}
	if info.EmailVerified != "true" {
		return nil, fmt.Errorf("google email is not verified")
	}

	return &info, nil
}

type googleAuthRequest struct {
	Credential string `json:"credential"`
}

// POST /api/auth/google — logs in an existing customer or creates a new one from a Google account.
func handleGoogleAuth(w http.ResponseWriter, r *http.Request) {
	var req googleAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	info, err := verifyGoogleIDToken(req.Credential)
	if err != nil {
		http.Error(w, "google sign-in failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	var customerID int
	err = db.QueryRow(`SELECT id FROM customers WHERE google_id = $1 OR email = $2`, info.Sub, info.Email).Scan(&customerID)
	if err != nil {
		// No existing account — create one.
		err = db.QueryRow(
			`INSERT INTO customers (full_name, email, phone, password_hash, google_id)
			 VALUES ($1, $2, '', '', $3) RETURNING id`,
			info.Name, info.Email, info.Sub,
		).Scan(&customerID)
		if err != nil {
			http.Error(w, "failed to create account", http.StatusInternalServerError)
			return
		}
	} else {
		// Existing account (maybe created via email/password before) — link the Google id if missing.
		db.Exec(`UPDATE customers SET google_id = $1 WHERE id = $2 AND google_id IS NULL`, info.Sub, customerID)
	}

	token, err := makeToken(customerID, "customer")
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"token": token, "customer_id": customerID, "full_name": info.Name, "email": info.Email})
}
