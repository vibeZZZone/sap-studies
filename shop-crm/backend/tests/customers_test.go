package tests

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
)

type customerResponse struct {
	ID            int    `json:"id"`
	CardNumber    string `json:"cardNumber"`
	Name          string `json:"name"`
	Phone         string `json:"phone"`
	BonusBalance  int    `json:"bonusBalance"`
	TotalSpentKop int64  `json:"totalSpentKop"`
}

func TestCustomersLifecycle(t *testing.T) {
	rec, _ := doRequest(t, "POST", "/api/customers", map[string]any{
		"name": "Пётр Иванов", "phone": "+7 999 555-11-22",
	})
	expectStatus(t, rec, 201)
	var created customerResponse
	decodeBody(t, rec, &created)

	if !strings.HasPrefix(created.CardNumber, "5000-") || len(created.CardNumber) != len("5000-0000") {
		t.Fatalf("card number %q does not follow the 5000-0001 format", created.CardNumber)
	}
	if created.BonusBalance != 0 {
		t.Fatalf("a new card should start with zero bonuses, got %d", created.BonusBalance)
	}

	// A second card gets the next number in the sequence.
	rec, _ = doRequest(t, "POST", "/api/customers", map[string]any{
		"name": "Анна Петрова", "phone": "+7 999 555-33-44",
	})
	expectStatus(t, rec, 201)
	var second customerResponse
	decodeBody(t, rec, &second)
	if second.CardNumber <= created.CardNumber {
		t.Fatalf("card numbers should increase: %q then %q", created.CardNumber, second.CardNumber)
	}

	rec, _ = doRequest(t, "GET", "/api/customers?search="+url.QueryEscape(second.Phone), nil)
	expectStatus(t, rec, 200)
	var found []customerResponse
	decodeBody(t, rec, &found)
	if len(found) != 1 || found[0].ID != second.ID {
		t.Fatalf("search by phone should return exactly the new customer, got %+v", found)
	}

	rec, _ = doRequest(t, "GET", "/api/customers?search="+second.CardNumber, nil)
	decodeBody(t, rec, &found)
	if len(found) != 1 || found[0].ID != second.ID {
		t.Fatalf("search by card number failed: %+v", found)
	}

	rec, _ = doRequest(t, "GET", "/api/customers?search="+strings.ToLower("пётр"), nil)
	decodeBody(t, rec, &found)
	if len(found) != 1 || found[0].ID != created.ID {
		t.Fatalf("case-insensitive search by name failed: %+v", found)
	}

	rec, _ = doRequest(t, "GET", "/api/customers", nil)
	expectStatus(t, rec, 200)
	var all []customerResponse
	decodeBody(t, rec, &all)
	if len(all) < 3 {
		t.Fatalf("expected at least the three demo customers plus new ones, got %d", len(all))
	}
}

// TestCustomerDuplicatePhone is the 409 branch: the phone is the natural key.
func TestCustomerDuplicatePhone(t *testing.T) {
	phone := fmt.Sprintf("+7 999 %s", "77-77-77")
	rec, _ := doRequest(t, "POST", "/api/customers", map[string]any{
		"name": "Первый", "phone": phone,
	})
	expectStatus(t, rec, 201)

	rec, env := doRequest(t, "POST", "/api/customers", map[string]any{
		"name": "Второй", "phone": phone,
	})
	expectStatus(t, rec, 409)
	if env.Error.Code != "PHONE_ALREADY_EXISTS" {
		t.Fatalf("expected PHONE_ALREADY_EXISTS, got %q", env.Error.Code)
	}
	if env.Error.Message == "" {
		t.Fatal("expected a human readable message")
	}
}

func TestCustomerValidation(t *testing.T) {
	cases := []struct {
		name string
		body any
	}{
		{"empty name", map[string]any{"name": "  ", "phone": "+79990001122"}},
		{"empty phone", map[string]any{"name": "Кто-то", "phone": ""}},
		{"phone too short", map[string]any{"name": "Кто-то", "phone": "12345"}},
		{"phone with letters", map[string]any{"name": "Кто-то", "phone": "+7 9ab 000 11 22"}},
		{"missing name", map[string]any{"phone": "+79990002222"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, _ := doRequest(t, "POST", "/api/customers", tc.body)
			expectStatus(t, rec, 400)
		})
	}
}