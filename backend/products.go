package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// GET /api/products
func handleListProducts(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, name, scent_notes, description, price_kobo, image_url, in_stock FROM products ORDER BY id`)
	if err != nil {
		http.Error(w, "failed to load products", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.ScentNotes, &p.Description, &p.PriceKobo, &p.ImageURL, &p.InStock); err != nil {
			http.Error(w, "failed to read products", http.StatusInternalServerError)
			return
		}
		products = append(products, p)
	}

	writeJSON(w, http.StatusOK, products)
}

// GET /api/products/{id}
func handleGetProduct(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/products/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid product id", http.StatusBadRequest)
		return
	}

	var p Product
	err = db.QueryRow(
		`SELECT id, name, scent_notes, description, price_kobo, image_url, in_stock FROM products WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.Name, &p.ScentNotes, &p.Description, &p.PriceKobo, &p.ImageURL, &p.InStock)
	if err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, p)
}

type productWriteRequest struct {
	Name        string `json:"name"`
	ScentNotes  string `json:"scent_notes"`
	Description string `json:"description"`
	PriceKobo   int    `json:"price_kobo"`
	ImageURL    string `json:"image_url"`
	InStock     *bool  `json:"in_stock"`
}

// POST /api/admin/products  (requires admin auth) — create a new product
func handleAdminCreateProduct(w http.ResponseWriter, r *http.Request, adminID int) {
	var req productWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.PriceKobo <= 0 {
		http.Error(w, "name and a positive price_kobo are required", http.StatusBadRequest)
		return
	}
	inStock := true
	if req.InStock != nil {
		inStock = *req.InStock
	}

	var id int
	err := db.QueryRow(
		`INSERT INTO products (name, scent_notes, description, price_kobo, image_url, in_stock)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		req.Name, req.ScentNotes, req.Description, req.PriceKobo, req.ImageURL, inStock,
	).Scan(&id)
	if err != nil {
		http.Error(w, "failed to create product", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

// PUT /api/admin/products/{id}  (requires admin auth) — edit an existing product
func handleAdminUpdateProduct(w http.ResponseWriter, r *http.Request, adminID int) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/products/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid product id", http.StatusBadRequest)
		return
	}

	var req productWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	inStock := true
	if req.InStock != nil {
		inStock = *req.InStock
	}

	_, err = db.Exec(
		`UPDATE products SET name = $1, scent_notes = $2, description = $3, price_kobo = $4, image_url = $5, in_stock = $6 WHERE id = $7`,
		req.Name, req.ScentNotes, req.Description, req.PriceKobo, req.ImageURL, inStock, id,
	)
	if err != nil {
		http.Error(w, "failed to update product", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"updated": true})
}

// DELETE /api/admin/products/{id}  (requires admin auth)
func handleAdminDeleteProduct(w http.ResponseWriter, r *http.Request, adminID int) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/products/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid product id", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(`DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		http.Error(w, "failed to delete product — it may have existing orders attached", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// handleAdminProductByID dispatches PUT/DELETE on /api/admin/products/{id} to the right handler.
func handleAdminProductByID(w http.ResponseWriter, r *http.Request, adminID int) {
	switch r.Method {
	case http.MethodPut:
		handleAdminUpdateProduct(w, r, adminID)
	case http.MethodDelete:
		handleAdminDeleteProduct(w, r, adminID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
