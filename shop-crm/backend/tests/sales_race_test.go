package tests

import (
	"sync"
	"testing"
)

// TestConcurrentSalesOnLastUnit is acceptance criterion 4: two parallel requests for
// the same last unit must not both succeed. Row locking in the sales transaction is
// what makes that hold.
func TestConcurrentSalesOnLastUnit(t *testing.T) {
	productID := seedProduct(t, "Гоночный товар (1 шт.)", 10000, 1)

	statuses := make([]int, 2)
	parallel(
		func() { statuses[0] = postSale(t, productID, 1) },
		func() { statuses[1] = postSale(t, productID, 1) },
	)

	created, conflicts := 0, 0
	for _, s := range statuses {
		switch s {
		case 201:
			created++
		case 409:
			conflicts++
		default:
			t.Fatalf("unexpected status %d in %v", s, statuses)
		}
	}
	if created != 1 || conflicts != 1 {
		t.Fatalf("expected exactly one 201 and one 409, got %v", statuses)
	}
	if stock := productStock(t, productID); stock != 0 {
		t.Fatalf("stock should be 0 after selling the last unit, got %d", stock)
	}
}

// TestConcurrentSalesDoNotOversell checks the same invariant with more units and more
// requests: the accepted quantity never exceeds the stock.
func TestConcurrentSalesDoNotOversell(t *testing.T) {
	const stock = 5
	const requests = 8

	productID := seedProduct(t, "Гоночный товар (5 шт.)", 5000, stock)

	var (
		mu       sync.Mutex
		accepted int
	)
	record := func(status int) {
		if status != 201 {
			return
		}
		mu.Lock()
		accepted++
		mu.Unlock()
	}

	fns := make([]func(), 0, requests)
	for i := 0; i < requests; i++ {
		fns = append(fns, func() { record(postSale(t, productID, 1)) })
	}
	parallel(fns...)

	if accepted != stock {
		t.Fatalf("expected %d accepted sales, got %d", stock, accepted)
	}
	if remaining := productStock(t, productID); remaining != 0 {
		t.Fatalf("expected stock 0, got %d", remaining)
	}
}

// TestConcurrentBonusUpdatesKeepBalanceConsistent makes sure the balance never goes
// negative when several sales for one card run at the same time.
func TestConcurrentBonusUpdatesKeepBalanceConsistent(t *testing.T) {
	productID := seedProduct(t, "Товар для гонки бонусов", 10000, 50)
	customerID := seedCustomer(t, "+7 901 000-00-01", 300)

	// Each sale asks for 50 % of 1000 ₽ = 500 bonuses while the card holds 300, so
	// the balance goes down to zero but never below.
	parallel(func() { postBonusSale(t, customerID, 50, productID) },
		func() { postBonusSale(t, customerID, 50, productID) })

	if bonus := customerBonus(t, customerID); bonus < 0 {
		t.Fatalf("bonus balance went negative: %d", bonus)
	}
}