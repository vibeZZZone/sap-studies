package tests

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

// seedProduct inserts a product directly and returns its id. Tests own their rows
// instead of depending on the shared demo seed.
func seedProduct(t *testing.T, name string, priceKop, stock int) int {
	t.Helper()
	var id int
	err := db.Pool.QueryRow(context.Background(),
		`INSERT INTO products (name, price_kop, stock) VALUES ($1, $2, $3) RETURNING id`,
		name, priceKop, stock).Scan(&id)
	if err != nil {
		t.Fatalf("seed product %q: %v", name, err)
	}
	return id
}

func seedCustomer(t *testing.T, phone string, bonusBalance int) int {
	t.Helper()
	var id int
	err := db.Pool.QueryRow(context.Background(),
		`INSERT INTO customers (card_number, name, phone, bonus_balance)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		uniqueCard(t), "Тест "+phone, phone, bonusBalance).Scan(&id)
	if err != nil {
		t.Fatalf("seed customer %q: %v", phone, err)
	}
	return id
}

// cardSeq hands out unique card numbers for seeded customers.
var cardSeq atomic.Int64

func uniqueCard(t *testing.T) string {
	t.Helper()
	// Production card numbers come from a sequence; tests only need a free one.
	return "9000-" + digitsFromInt(int(cardSeq.Add(1)))
}

func productStock(t *testing.T, id int) int {
	t.Helper()
	var stock int
	if err := db.Pool.QueryRow(context.Background(), `SELECT stock FROM products WHERE id = $1`, id).Scan(&stock); err != nil {
		t.Fatalf("read stock of product %d: %v", id, err)
	}
	return stock
}

func customerBonus(t *testing.T, id int) int {
	t.Helper()
	var bonus int
	if err := db.Pool.QueryRow(context.Background(), `SELECT bonus_balance FROM customers WHERE id = $1`, id).Scan(&bonus); err != nil {
		t.Fatalf("read bonus of customer %d: %v", id, err)
	}
	return bonus
}

func customerTotalSpent(t *testing.T, id int) int64 {
	t.Helper()
	var total int64
	if err := db.Pool.QueryRow(context.Background(), `SELECT total_spent_kop FROM customers WHERE id = $1`, id).Scan(&total); err != nil {
		t.Fatalf("read total spent of customer %d: %v", id, err)
	}
	return total
}

func saleCount(t *testing.T, customerID int) int {
	t.Helper()
	var count int
	if err := db.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM sales WHERE customer_id = $1`, customerID).Scan(&count); err != nil {
		t.Fatalf("count sales of customer %d: %v", customerID, err)
	}
	return count
}

func postSale(t *testing.T, productID, qty int) int {
	t.Helper()
	rec, _ := doRequest(t, "POST", "/api/sales", map[string]any{
		"items": []map[string]any{{"productId": productID, "qty": qty}},
	})
	return rec.Code
}

func postBonusSale(t *testing.T, customerID, percent, productID int) int {
	t.Helper()
	rec, _ := doRequest(t, "POST", "/api/sales", map[string]any{
		"customerId":   customerID,
		"bonusPercent": percent,
		"items":        []map[string]any{{"productId": productID, "qty": 1}},
	})
	return rec.Code
}

// parallel runs fns at the same time and waits for all of them. Each fn must not call
// t.Fatal directly; tests pass status codes back through shared variables instead.
func parallel(fns ...func()) {
	var wg sync.WaitGroup
	wg.Add(len(fns))
	for _, fn := range fns {
		go func(f func()) {
			defer wg.Done()
			f()
		}(fn)
	}
	wg.Wait()
}

func digitsFromInt(n int) string {
	const alphabet = "0123456789"
	if n == 0 {
		return "0000"
	}
	out := make([]byte, 0, 8)
	for n > 0 {
		out = append([]byte{alphabet[n%10]}, out...)
		n /= 10
	}
	for len(out) < 4 {
		out = append([]byte{'0'}, out...)
	}
	return string(out)
}