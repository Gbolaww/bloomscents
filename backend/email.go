package main

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
)

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

	subject := "New Bloom Scents order — " + order.ProductName
	body := fmt.Sprintf(
		"New paid order!\n\nOrder ID: %d\nProduct: %s\nQuantity: %d\nAmount: %.2f NGN\nReference: %s\nDelivery address: %s\n",
		order.ID, order.ProductName, order.Quantity, float64(order.AmountKobo)/100, order.PaystackReference, order.DeliveryAddress,
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
