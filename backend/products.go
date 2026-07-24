package main

import (
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
