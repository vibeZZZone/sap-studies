package tests

import "testing"

type productResponse struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	PriceKop int64   `json:"priceKop"`
	Price    float64 `json:"price"`
	Stock    int     `json:"stock"`
	Archived bool    `json:"archived"`
	LowStock bool    `json:"lowStock"`
}

func TestProductsLifecycle(t *testing.T) {
	rec, _ := doRequest(t, "POST", "/api/products", map[string]any{
		"name": "  Новый товар  ", "price": "349.50", "initialQty": 4,
	})
	expectStatus(t, rec, 201)
	var created productResponse
	decodeBody(t, rec, &created)

	if created.Name != "Новый товар" {
		t.Fatalf("name should be trimmed, got %q", created.Name)
	}
	if created.PriceKop != 34950 || created.Price != 349.5 {
		t.Fatalf("price should be 34950 kop, got %d (%v)", created.PriceKop, created.Price)
	}
	if !created.LowStock {
		t.Fatal("stock of 4 should be flagged as low")
	}

	// Acceptance criterion: the restock endpoint takes an arbitrary quantity.
	rec, _ = doRequest(t, "POST", "/api/products/"+itoa(created.ID)+"/restock", map[string]any{"qty": 10})
	expectStatus(t, rec, 200)
	var restocked productResponse
	decodeBody(t, rec, &restocked)
	if restocked.Stock != 14 || restocked.LowStock {
		t.Fatalf("after restock expected stock 14 and no low flag, got %d low=%v", restocked.Stock, restocked.LowStock)
	}

	rec, _ = doRequest(t, "GET", "/api/products", nil)
	expectStatus(t, rec, 200)
	var list []productResponse
	decodeBody(t, rec, &list)

	var found bool
	for _, p := range list {
		if p.ID == created.ID {
			found = true
		}
		if p.Archived {
			t.Fatalf("archived product %d leaked into the list", p.ID)
		}
	}
	if !found {
		t.Fatal("created product is missing from the list")
	}

	// Soft delete keeps the row, so old receipts still resolve the product.
	rec, _ = doRequest(t, "DELETE", "/api/products/"+itoa(created.ID), nil)
	expectStatus(t, rec, 204)
	rec, _ = doRequest(t, "DELETE", "/api/products/"+itoa(created.ID), nil)
	expectStatus(t, rec, 404)

	rec, _ = doRequest(t, "GET", "/api/products", nil)
	decodeBody(t, rec, &list)
	for _, p := range list {
		if p.ID == created.ID {
			t.Fatal("archived product should not appear in the list")
		}
	}
	var archivedFlag bool
	if err := db.Pool.QueryRow(t.Context(), `SELECT archived FROM products WHERE id = $1`, created.ID).Scan(&archivedFlag); err != nil {
		t.Fatalf("read archived flag: %v", err)
	}
	if !archivedFlag {
		t.Fatal("the row should still exist with archived = true")
	}
}

func TestProductsValidation(t *testing.T) {
	cases := []struct {
		name string
		body any
	}{
		{"empty name", map[string]any{"name": "   ", "price": "10", "initialQty": 1}},
		{"zero price", map[string]any{"name": "X", "price": "0", "initialQty": 1}},
		{"negative price", map[string]any{"name": "X", "price": "-5", "initialQty": 1}},
		{"three decimals", map[string]any{"name": "X", "price": "1.234", "initialQty": 1}},
		{"garbage price", map[string]any{"name": "X", "price": "abc", "initialQty": 1}},
		{"negative stock", map[string]any{"name": "X", "price": "10", "initialQty": -1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, _ := doRequest(t, "POST", "/api/products", tc.body)
			expectStatus(t, rec, 400)
		})
	}

	id := seedProduct(t, "Товар для проверки остатка", 1000, 5)
	rec, _ := doRequest(t, "POST", "/api/products/"+itoa(id)+"/restock", map[string]any{"qty": 0})
	expectStatus(t, rec, 400)
	rec, _ = doRequest(t, "POST", "/api/products/99999999/restock", map[string]any{"qty": 5})
	expectStatus(t, rec, 404)
	rec, _ = doRequest(t, "POST", "/api/products/abc/restock", map[string]any{"qty": 5})
	expectStatus(t, rec, 400)
	if stock := productStock(t, id); stock != 5 {
		t.Fatalf("stock must stay 5, got %d", stock)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var out []byte
	for n > 0 {
		out = append([]byte{byte('0' + n%10)}, out...)
		n /= 10
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}