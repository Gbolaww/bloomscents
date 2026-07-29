package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// sendEmailViaResend sends an email using the Resend HTTPS API instead of raw SMTP.
// Raw SMTP (port 587/465) is blocked outbound on many cloud hosts including Railway,
// so we send over HTTPS instead — this avoids that restriction entirely.
//
// Configure:
//
//	RESEND_API_KEY  — from resend.com (free tier: 3,000 emails/month)
//	RESEND_FROM     — the "from" address. Until you verify your own domain on Resend,
//	                  use 'onboarding@resend.dev' (Resend's shared test sender).
func sendEmailViaResend(to, subject, body string) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	from := os.Getenv("RESEND_FROM")
	if from == "" {
		from = "onboarding@resend.dev"
	}
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY is not set")
	}

	payload := map[string]any{
		"from":    from,
		"to":      []string{to},
		"subject": subject,
		"text":    body,
	}
	bodyBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("resend returned status %d", resp.StatusCode)
	}
	return nil
}

// sendCustomerShippedNotification emails the customer when their order is marked shipped.
func sendCustomerShippedNotification(order Order) {
	if order.CustomerEmail == "" {
		log.Println("customer has no email; skipping shipped notification")
		return
	}

	itemsSummary := ""
	for _, item := range order.Items {
		itemsSummary += fmt.Sprintf("  - %s x%d\n", item.ProductName, item.Quantity)
	}

	subject := "Your Bloom Scents order is on its way!"
	body := fmt.Sprintf(
		"Hi %s,\n\nGreat news — your order has been shipped and is on its way to you.\n\nItems:\n%s\nDelivery address: %s\n\nThank you for shopping with Bloom Scents!\n",
		order.CustomerName, itemsSummary, order.DeliveryAddress,
	)

	if err := sendEmailViaResend(order.CustomerEmail, subject, body); err != nil {
		log.Printf("failed to send customer shipped notification: %v", err)
	}
}

// sendAdminOrderNotification emails the admin whenever an order is confirmed paid.
func sendAdminOrderNotification(order Order) {
	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		log.Println("ADMIN_EMAIL not set; skipping admin email notification")
		return
	}

	itemsSummary := ""
	for _, item := range order.Items {
		itemsSummary += fmt.Sprintf("  - %s x%d (%.2f NGN each)\n", item.ProductName, item.Quantity, float64(item.PriceKobo)/100)
	}

	subject := fmt.Sprintf("New Bloom Scents order — #%d", order.ID)
	body := fmt.Sprintf(
		"New paid order!\n\nOrder ID: %d\nItems:\n%s\nTotal: %.2f NGN\nReference: %s\nDelivery address: %s\n",
		order.ID, itemsSummary, float64(order.TotalAmountKobo)/100, order.PaystackReference, order.DeliveryAddress,
	)

	if err := sendEmailViaResend(adminEmail, subject, body); err != nil {
		log.Printf("failed to send admin notification email: %v", err)
	}
}
