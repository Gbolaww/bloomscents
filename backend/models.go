package main

import "time"

type Customer struct {
	ID              int       `json:"id"`
	FullName        string    `json:"full_name"`
	Email           string    `json:"email"`
	Phone           string    `json:"phone"`
	DeliveryAddress string    `json:"delivery_address"`
	CreatedAt       time.Time `json:"created_at"`
}

type Admin struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
}

type Product struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ScentNotes  string `json:"scent_notes"`
	Description string `json:"description"`
	PriceKobo   int    `json:"price_kobo"`
	ImageURL    string `json:"image_url"`
	InStock     bool   `json:"in_stock"`
}

type OrderItem struct {
	ProductID   int    `json:"product_id"`
	ProductName string `json:"product_name,omitempty"`
	Quantity    int    `json:"quantity"`
	PriceKobo   int    `json:"price_kobo"`
}

type Order struct {
	ID                int         `json:"id"`
	CustomerID        int         `json:"customer_id"`
	Items             []OrderItem `json:"items,omitempty"`
	TotalAmountKobo   int         `json:"total_amount_kobo"`
	PaystackReference string      `json:"paystack_reference"`
	Status            string      `json:"status"`
	DeliveryAddress   string      `json:"delivery_address"`
	CreatedAt         time.Time   `json:"created_at"`
}
