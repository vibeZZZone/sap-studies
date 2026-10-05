package tests

import (
	"testing"
)

type saleResponse struct {
	ID          int    `json:"id"`
	SubtotalKop int64  `json:"subtotalKop"`
	Subtotal    string `json:"subtotal"`
	BonusSpent  int    `json:"bonusSpent"`
	TotalKop    int64  `json:"totalKop"`
	Total       string `json:"total"`
	BonusEarned int    `json:"bonusEarned"`
	CustomerID  *int   `json:"customerId"`
	Customer    *struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		CardNumber string `json:"cardNumber"`
	} `json:"customer"`
	Items []struct {
		ProductID    int    `json:"productId"`
		Name         string `json:"name"`
		PriceKop     int64  `json:"priceKop"`
		Price        string `json:"price"`
		Qty          int    `json:"qty"`
		SumKop       int64  `json:"sumKop"`
	} `json:"items"`
	CreatedAt string `json:"createdAt"`
}

// TestSaleWithoutCard: no card means no bonuses at all, even when the client asks.
func TestSaleWithoutCard(t *testing.T) {
	productID := seedProduct(t, "Товар без карты", 20000, 10)
	stockBefore := productStock(t, productID)

	rec, _ := doRequest(t, "POST", "/api/sales", map[string]any{
		"bonusPercent": 50,
		"items":        []map[string]any{{"productId": productID, "qty": 2}},
	})
	expectStatus(t, rec, 201)

	var sale saleResponse
	decodeBody(t, rec, &sale)

	if sale.BonusSpent != 0 || sale.BonusEarned != 0 {
		t.Fatalf("expected no bonuses without a card, got spent=%d earned=%d", sale.BonusSpent, sale.BonusEarned)
	}
	if sale.TotalKop != sale.SubtotalKop {
		t.Fatalf("total %d should equal subtotal %d", sale.TotalKop, sale.SubtotalKop)
	}
	if sale.CustomerID != nil || sale.Customer != nil {
		t.Fatalf("expected no customer on the sale, got %+v", sale.Customer)
	}
	if got, want := productStock(t, productID), stockBefore-2; got != want {
		t.Fatalf("stock should be %d, got %d", want, got)
	}
}

// TestSaleWithBonusMatchesAcceptanceExample is acceptance criterion 2: 1000 ₽ with
// 800 bonuses on the card, half spent by bonuses.
func TestSaleWithBonusMatchesAcceptanceExample(t *testing.T) {
	productID := seedProduct(t, "Товар для примера", 50000, 10)
	customerID := seedCustomer(t, "+7 902 000-00-02", 800)

	rec, _ := doRequest(t, "POST", "/api/sales", map[string]any{
		"customerId":   customerID,
		"bonusPercent": 50,
		"items":        []map[string]any{{"productId": productID, "qty": 2}},
	})
	expectStatus(t, rec, 201)

	var sale saleResponse
	decodeBody(t, rec, &sale)

	if sale.SubtotalKop != 100000 {
		t.Fatalf("subtotal should be 100000 kop, got %d", sale.SubtotalKop)
	}
	if sale.BonusSpent != 500 {
		t.Fatalf("bonuses spent should be 500, got %d", sale.BonusSpent)
	}
	if sale.TotalKop != 50000 {
		t.Fatalf("total should be 50000 kop, got %d", sale.TotalKop)
	}
	if sale.BonusEarned != 25 {
		t.Fatalf("bonuses earned should be 25, got %d", sale.BonusEarned)
	}
	if sale.Customer == nil || sale.Customer.CardNumber == "" {
		t.Fatal("expected the sale to reference a customer card")
	}

	// Balance: 800 − 500 spent + 25 earned.
	if bonus := customerBonus(t, customerID); bonus != 325 {
		t.Fatalf("bonus balance should be 325, got %d", bonus)
	}
	if spent := customerTotalSpent(t, customerID); spent != 50000 {
		t.Fatalf("total spent should be 50000 kop, got %d", spent)
	}
}

// TestSaleClampsBonusToBalance: a request above the balance is trimmed, not rejected.
func TestSaleClampsBonusToBalance(t *testing.T) {
	productID := seedProduct(t, "Товар для проверки баланса", 50000, 10)
	customerID := seedCustomer(t, "+7 902 000-00-03", 120)

	rec, _ := doRequest(t, "POST", "/api/sales", map[string]any{
		"customerId":   customerID,
		"bonusPercent": 50,
		"items":        []map[string]any{{"productId": productID, "qty": 1}},
	})
	expectStatus(t, rec, 201)

	var sale saleResponse
	decodeBody(t, rec, &sale)
	// A balance of 120 points is 120 ₽, which is below the 250 ₽ cap.
	if sale.BonusSpent != 120 {
		t.Fatalf("bonuses spent should be clamped to 120, got %d", sale.BonusSpent)
	}
	if sale.TotalKop != 50000-12000 {
		t.Fatalf("total should be %d kop, got %d", 50000-12000, sale.TotalKop)
	}
	if bonus := customerBonus(t, customerID); bonus != sale.BonusEarned {
		t.Fatalf("balance should equal the newly earned bonus %d, got %d", sale.BonusEarned, bonus)
	}
}

// TestSaleInsufficientStockIsConflict is acceptance criterion 3: a 409 that changes
// neither stock nor bonuses.
func TestSaleInsufficientStockIsConflict(t *testing.T) {
	productID := seedProduct(t, "Мало товара", 10000, 1)
	customerID := seedCustomer(t, "+7 902 000-00-04", 500)
	salesBefore := saleCount(t, customerID)

	rec, env := doRequest(t, "POST", "/api/sales", map[string]any{
		"customerId":   customerID,
		"bonusPercent": 50,
		"items":        []map[string]any{{"productId": productID, "qty": 3}},
	})
	expectStatus(t, rec, 409)
	if env.Error.Code != "INSUFFICIENT_STOCK" {
		t.Fatalf("expected INSUFFICIENT_STOCK, got %q", env.Error.Code)
	}
	if env.Error.Params["productId"] == "" {
		t.Fatalf("expected the conflicting product id in params, got %+v", env.Error.Params)
	}

	if stock := productStock(t, productID); stock != 1 {
		t.Fatalf("stock must stay 1, got %d", stock)
	}
	if bonus := customerBonus(t, customerID); bonus != 500 {
		t.Fatalf("bonus balance must stay 500, got %d", bonus)
	}
	if count := saleCount(t, customerID); count != salesBefore {
		t.Fatalf("no sale should be recorded, count went from %d to %d", salesBefore, count)
	}
}

// TestSaleUnknownProductAndCustomer covers the 404 branches.
func TestSaleUnknownProductAndCustomer(t *testing.T) {
	rec, env := doRequest(t, "POST", "/api/sales", map[string]any{
		"items": []map[string]any{{"productId": 99999999, "qty": 1}},
	})
	expectStatus(t, rec, 404)
	if env.Error.Code != "PRODUCT_NOT_FOUND" {
		t.Fatalf("expected PRODUCT_NOT_FOUND, got %q", env.Error.Code)
	}

	rec, env = doRequest(t, "POST", "/api/sales", map[string]any{
		"customerId": 99999999,
		"items":      []map[string]any{{"productId": seedProduct(t, "Товар для 404 покупателя", 1000, 5), "qty": 1}},
	})
	expectStatus(t, rec, 404)
	if env.Error.Code != "CUSTOMER_NOT_FOUND" {
		t.Fatalf("expected CUSTOMER_NOT_FOUND, got %q", env.Error.Code)
	}
}

// TestSaleValidation covers the 400 branch.
func TestSaleValidation(t *testing.T) {
	productID := seedProduct(t, "Товар для валидации", 10000, 5)

	cases := []struct {
		name string
		body any
		code string
	}{
		{"empty cart", map[string]any{"items": []map[string]any{}}, "EMPTY_CART"},
		{"zero qty", map[string]any{"items": []map[string]any{{"productId": productID, "qty": 0}}}, "VALIDATION_ERROR"},
		{"negative qty", map[string]any{"items": []map[string]any{{"productId": productID, "qty": -1}}}, "VALIDATION_ERROR"},
		{"percent above limit", map[string]any{"bonusPercent": 80, "items": []map[string]any{{"productId": productID, "qty": 1}}}, "VALIDATION_ERROR"},
		{"negative percent", map[string]any{"bonusPercent": -5, "items": []map[string]any{{"productId": productID, "qty": 1}}}, "VALIDATION_ERROR"},
		{"duplicate product", map[string]any{"items": []map[string]any{{"productId": productID, "qty": 1}, {"productId": productID, "qty": 1}}}, "VALIDATION_ERROR"},
		{"unknown field", map[string]any{"items": []map[string]any{{"productId": productID, "qty": 1}}, "surprise": true}, "VALIDATION_ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, env := doRequest(t, "POST", "/api/sales", tc.body)
			expectStatus(t, rec, 400)
			if env.Error.Code != tc.code {
				t.Fatalf("expected %q, got %q", tc.code, env.Error.Code)
			}
		})
	}
}

// TestPriceChangeDoesNotRewriteHistory is acceptance criterion 6.
func TestPriceChangeDoesNotRewriteHistory(t *testing.T) {
	productID := seedProduct(t, "Товар со сменой цены", 10000, 10)

	rec, _ := doRequest(t, "POST", "/api/sales", map[string]any{
		"items": []map[string]any{{"productId": productID, "qty": 1}},
	})
	expectStatus(t, rec, 201)
	var sale saleResponse
	decodeBody(t, rec, &sale)
	originalTotal := sale.TotalKop

	if _, err := db.Pool.Exec(t.Context(), `UPDATE products SET price_kop = 99999 WHERE id = $1`, productID); err != nil {
		t.Fatalf("change price: %v", err)
	}

	rec, _ = doRequest(t, "GET", "/api/sales?page=1", nil)
	expectStatus(t, rec, 200)
	var page struct {
		Sales []saleResponse `json:"sales"`
	}
	decodeBody(t, rec, &page)

	var found bool
	for _, s := range page.Sales {
		if s.ID != sale.ID {
			continue
		}
		found = true
		if s.TotalKop != originalTotal {
			t.Fatalf("historical total changed: %d instead of %d", s.TotalKop, originalTotal)
		}
		if len(s.Items) != 1 || s.Items[0].PriceKop != 10000 {
			t.Fatalf("historical line price changed: %+v", s.Items)
		}
		if s.Items[0].Name == "" {
			t.Fatal("expected a name snapshot on the sale line")
		}
	}
	if !found {
		t.Fatalf("sale %d not found in the history", sale.ID)
	}
}

// TestSalesPagination checks the 20-per-page rule and the newest-first order.
func TestSalesPagination(t *testing.T) {
	productID := seedProduct(t, "Товар для пагинации", 1000, 100)
	for i := 0; i < 25; i++ {
		if code := postSale(t, productID, 1); code != 201 {
			t.Fatalf("seeding sale %d failed with %d", i, code)
		}
	}

	rec, _ := doRequest(t, "GET", "/api/sales?page=1", nil)
	expectStatus(t, rec, 200)
	var page struct {
		Sales    []saleResponse `json:"sales"`
		Page     int            `json:"page"`
		PageSize int            `json:"pageSize"`
		Total    int            `json:"total"`
		Pages    int            `json:"pages"`
	}
	decodeBody(t, rec, &page)

	if page.PageSize != 20 || len(page.Sales) != 20 {
		t.Fatalf("expected 20 sales on the first page, got pageSize=%d len=%d", page.PageSize, len(page.Sales))
	}
	if page.Total < 25 {
		t.Fatalf("expected at least 25 sales in total, got %d", page.Total)
	}
	if page.Pages != (page.Total+19)/20 {
		t.Fatalf("pages %d does not match total %d", page.Pages, page.Total)
	}
	for i := 1; i < len(page.Sales); i++ {
		if page.Sales[i-1].ID < page.Sales[i].ID {
			t.Fatalf("sales are not ordered newest first: %d before %d", page.Sales[i-1].ID, page.Sales[i].ID)
		}
	}
	for _, s := range page.Sales {
		if len(s.Items) == 0 {
			t.Fatalf("sale %d came back without items", s.ID)
		}
	}

	rec, _ = doRequest(t, "GET", "/api/sales?page=0", nil)
	expectStatus(t, rec, 400)
}