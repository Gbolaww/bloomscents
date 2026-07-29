package main

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
)

// sendCustomerShippedNotification emails the customer when their order is marked shipped.
// Uses the same SMTP_* environment variables as the admin notification.
func sendCustomerShippedNotification(order Order) {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASSWORD")

	if host == "" || order.CustomerEmail == "" {
		log.Println("SMTP not configured or customer has no email; skipping shipped notification")
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

	msg := []byte("To: " + order.CustomerEmail + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" + body + "\r\n")

	auth := smtp.PlainAuth("", user, password, host)
	addr := host + ":" + port

	if err := smtp.SendMail(addr, auth, user, []string{order.CustomerEmail}, msg); err != nil {
		log.Printf("failed to send customer shipped notification: %v", err)
	}
}

// sendAdminOrderNotification emails the admin whenever an order is confirmed paid.
// Configure these environment variables:
//
//	SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD, ADMIN_EMAIL
//
// For Gmail: SMTP_HOST=smtp.gmail.com, SMTP_PORT=587, SMTP_USER=you@gmail.com,
// SMTP_PASSWORD=<app password, not your normal password>
func sendAdminOrderNotification(order Order) {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASSWORD")
	adminEmail := os.Getenv("ADMIN_EMAIL")

	if host == "" || adminEmail == "" {
		log.Println("SMTP not configured; skipping admin email notification")
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

	msg := []byte("To: " + adminEmail + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" + body + "\r\n")

	auth := smtp.PlainAuth("", user, password, host)
	addr := host + ":" + port

	if err := smtp.SendMail(addr, auth, user, []string{adminEmail}, msg); err != nil {
		log.Printf("failed to send admin notification email: %v", err)
	}
}
