package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const paystackBaseURL = "https://api.paystack.co"

type paystackInitResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		AuthorizationURL string `json:"authorization_url"`
		AccessCode       string `json:"access_code"`
		Reference        string `json:"reference"`
	} `json:"data"`
}

// initializePaystackTransaction asks Paystack to create a payment session
// for the given amount (in kobo) and customer email. Returns the checkout URL and reference.
func initializePaystackTransaction(email string, amountKobo int, reference string) (checkoutURL string, err error) {
	secretKey := os.Getenv("PAYSTACK_SECRET_KEY")
	if secretKey == "" {
		return "", fmt.Errorf("PAYSTACK_SECRET_KEY is not set")
	}

	body := map[string]any{
		"email":     email,
		"amount":    amountKobo,
		"reference": reference,
	}
	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", paystackBaseURL+"/transaction/initialize", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+secretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var parsed paystackInitResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if !parsed.Status {
		return "", fmt.Errorf("paystack error: %s", parsed.Message)
	}

	return parsed.Data.AuthorizationURL, nil
}

// verifyPaystackWebhookSignature checks the x-paystack-signature header against
// the raw request body using HMAC-SHA512, per Paystack's webhook docs.
func verifyPaystackWebhookSignature(body []byte, signature string) bool {
	secretKey := os.Getenv("PAYSTACK_SECRET_KEY")
	mac := hmac.New(sha512.New, []byte(secretKey))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

type paystackWebhookPayload struct {
	Event string `json:"event"`
	Data  struct {
		Reference string `json:"reference"`
		Status    string `json:"status"`
		Amount    int    `json:"amount"`
		Customer  struct {
			Email string `json:"email"`
		} `json:"customer"`
	} `json:"data"`
}

// POST /api/paystack/webhook
func handlePaystackWebhook(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	signature := r.Header.Get("x-paystack-signature")
	if !verifyPaystackWebhookSignature(bodyBytes, signature) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var payload paystackWebhookPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if payload.Event == "charge.success" && payload.Data.Status == "success" {
		order, err := markOrderPaid(payload.Data.Reference)
		if err == nil {
			go sendAdminOrderNotification(order)
		}
	}

	w.WriteHeader(http.StatusOK)
}
