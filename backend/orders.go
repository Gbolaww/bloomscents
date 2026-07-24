package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
)

type createOrderRequest struct {
	ProductID       int    `json:"product_id"`
	Quantity        int    `json:"quantity"`
	DeliveryAddress string `json:"delivery_address"`
	Email           string `json:"email"` // used for the Paystack checkout session
}

// POST /api/orders  (requires customer auth)
func handleCreateOrder(w http.ResponseWriter, r *http.Request, customerID int) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Quantity < 1 {
		req.Quantity = 1
	}

	var priceKobo int
	err := db.QueryRow(`SELECT price_kobo FROM products WHERE id = $1 AND in_stock = TRUE`, req.ProductID).Scan(&priceKobo)
	if err != nil {
		http.Error(w, "product not found or out of stock", http.StatusNotFound)
		return
	}
	amountKobo := priceKobo * req.Quantity

	reference, err := generateReference()
	if err != nil {
		http.Error(w, "failed to create order reference", http.StatusInternalServerError)
		return
	}

	var orderID int
	err = db.QueryRow(
		`INSERT INTO orders (customer_id, product_id, quantity, amount_kobo, paystack_reference, status, delivery_address)
		 VALUES ($1, $2, $3, $4, $5, 'pending', $6) RETURNING id`,
		customerID, req.ProductID, req.Quantity, amountKobo, reference, req.DeliveryAddress,
	).Scan(&orderID)
	if err != nil {
		http.Error(w, "failed to create order", http.StatusInternalServerError)
		return
	}

	checkoutURL, err := initializePaystackTransaction(req.Email, amountKobo, reference)
	if err != nil {
		http.Error(w, "failed to start payment: "+err.Error(), http.StatusBadGateway)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"order_id":     orderID,
		"reference":    reference,
		"checkout_url": checkoutURL,
	})
}

// GET /api/orders/history  (requires customer auth)
func handleOrderHistory(w http.ResponseWriter, r *http.Request, customerID int) {
	rows, err := db.Query(
		`SELECT o.id, o.product_id, p.name, o.quantity, o.amount_kobo, o.paystack_reference, o.status, o.delivery_address, o.created_at
		 FROM orders o JOIN products p ON p.id = o.product_id
		 WHERE o.customer_id = $1 ORDER BY o.created_at DESC`,
		customerID,
	)
	if err != nil {
		http.Error(w, "failed to load orders", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.ProductID, &o.ProductName, &o.Quantity, &o.AmountKobo, &o.PaystackReference, &o.Status, &o.DeliveryAddress, &o.CreatedAt); err != nil {
			http.Error(w, "failed to read orders", http.StatusInternalServerError)
			return
		}
		orders = append(orders, o)
	}

	writeJSON(w, http.StatusOK, orders)
}

// GET /api/admin/orders  (requires admin auth)
func handleAdminListOrders(w http.ResponseWriter, r *http.Request, adminID int) {
	rows, err := db.Query(
		`SELECT o.id, o.customer_id, o.product_id, p.name, o.quantity, o.amount_kobo, o.paystack_reference, o.status, o.delivery_address, o.created_at
		 FROM orders o JOIN products p ON p.id = o.product_id
		 ORDER BY o.created_at DESC`,
	)
	if err != nil {
		http.Error(w, "failed to load orders", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.ProductID, &o.ProductName, &o.Quantity, &o.AmountKobo, &o.PaystackReference, &o.Status, &o.DeliveryAddress, &o.CreatedAt); err != nil {
			http.Error(w, "failed to read orders", http.StatusInternalServerError)
			return
		}
		orders = append(orders, o)
	}

	writeJSON(w, http.StatusOK, orders)
}

func markOrderPaid(reference string) (Order, error) {
	var o Order
	err := db.QueryRow(
		`UPDATE orders SET status = 'paid' WHERE paystack_reference = $1
		 RETURNING id, customer_id, product_id, quantity, amount_kobo, paystack_reference, status, delivery_address, created_at`,
		reference,
	).Scan(&o.ID, &o.CustomerID, &o.ProductID, &o.Quantity, &o.AmountKobo, &o.PaystackReference, &o.Status, &o.DeliveryAddress, &o.CreatedAt)
	if err != nil {
		return o, err
	}

	db.QueryRow(`SELECT name FROM products WHERE id = $1`, o.ProductID).Scan(&o.ProductName)
	return o, nil
}

func generateReference() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "bloom_" + hex.EncodeToString(b), nil
}
