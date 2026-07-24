package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
)

type cartItemRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type createOrderRequest struct {
	Items           []cartItemRequest `json:"items"`
	DeliveryAddress string            `json:"delivery_address"`
	Email           string            `json:"email"` // used for the Paystack checkout session
}

// POST /api/orders  (requires customer auth) — creates one order from a cart of items
func handleCreateOrder(w http.ResponseWriter, r *http.Request, customerID int) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.Items) == 0 {
		http.Error(w, "cart is empty", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "failed to start order", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	reference, err := generateReference()
	if err != nil {
		http.Error(w, "failed to create order reference", http.StatusInternalServerError)
		return
	}

	var orderID int
	err = tx.QueryRow(
		`INSERT INTO orders (customer_id, total_amount_kobo, paystack_reference, status, delivery_address)
		 VALUES ($1, 0, $2, 'pending', $3) RETURNING id`,
		customerID, reference, req.DeliveryAddress,
	).Scan(&orderID)
	if err != nil {
		http.Error(w, "failed to create order", http.StatusInternalServerError)
		return
	}

	totalKobo := 0
	for _, item := range req.Items {
		qty := item.Quantity
		if qty < 1 {
			qty = 1
		}
		var priceKobo int
		err := tx.QueryRow(`SELECT price_kobo FROM products WHERE id = $1 AND in_stock = TRUE`, item.ProductID).Scan(&priceKobo)
		if err != nil {
			http.Error(w, "one of the products is unavailable", http.StatusNotFound)
			return
		}
		_, err = tx.Exec(
			`INSERT INTO order_items (order_id, product_id, quantity, price_kobo) VALUES ($1, $2, $3, $4)`,
			orderID, item.ProductID, qty, priceKobo,
		)
		if err != nil {
			http.Error(w, "failed to add item to order", http.StatusInternalServerError)
			return
		}
		totalKobo += priceKobo * qty
	}

	if _, err := tx.Exec(`UPDATE orders SET total_amount_kobo = $1 WHERE id = $2`, totalKobo, orderID); err != nil {
		http.Error(w, "failed to finalize order total", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to save order", http.StatusInternalServerError)
		return
	}

	checkoutURL, err := initializePaystackTransaction(req.Email, totalKobo, reference)
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
	orders, err := loadOrdersWithItems(`WHERE o.customer_id = $1 ORDER BY o.created_at DESC`, customerID)
	if err != nil {
		http.Error(w, "failed to load orders", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

// GET /api/admin/orders  (requires admin auth)
func handleAdminListOrders(w http.ResponseWriter, r *http.Request, adminID int) {
	orders, err := loadOrdersWithItems(`ORDER BY o.created_at DESC`)
	if err != nil {
		http.Error(w, "failed to load orders", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

// loadOrdersWithItems loads orders (optionally filtered) plus their line items.
func loadOrdersWithItems(whereAndOrder string, args ...any) ([]Order, error) {
	rows, err := db.Query(
		`SELECT o.id, o.customer_id, o.total_amount_kobo, o.paystack_reference, o.status, o.delivery_address, o.created_at
		 FROM orders o `+whereAndOrder,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.TotalAmountKobo, &o.PaystackReference, &o.Status, &o.DeliveryAddress, &o.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}

	for i := range orders {
		itemRows, err := db.Query(
			`SELECT oi.product_id, p.name, oi.quantity, oi.price_kobo
			 FROM order_items oi JOIN products p ON p.id = oi.product_id
			 WHERE oi.order_id = $1`,
			orders[i].ID,
		)
		if err != nil {
			return nil, err
		}
		var items []OrderItem
		for itemRows.Next() {
			var it OrderItem
			if err := itemRows.Scan(&it.ProductID, &it.ProductName, &it.Quantity, &it.PriceKobo); err != nil {
				itemRows.Close()
				return nil, err
			}
			items = append(items, it)
		}
		itemRows.Close()
		orders[i].Items = items
	}

	return orders, nil
}

func markOrderPaid(reference string) (Order, error) {
	_, err := db.Exec(`UPDATE orders SET status = 'paid' WHERE paystack_reference = $1`, reference)
	if err != nil {
		return Order{}, err
	}
	orders, err := loadOrdersWithItems(`WHERE o.paystack_reference = $1`, reference)
	if err != nil || len(orders) == 0 {
		return Order{}, err
	}
	return orders[0], nil
}

func generateReference() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "bloom_" + hex.EncodeToString(b), nil
}
