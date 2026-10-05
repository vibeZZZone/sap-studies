package bonus

import "testing"

func id(n int) *int { return &n }

// TestCalculate covers the acceptance examples and the rounding edges: every
// division floors, and a sale without a card never touches bonuses.
//
// Units: subtotal and total are kopecks, bonus amounts are bonus points worth one
// ruble each.
func TestCalculate(t *testing.T) {
	tests := []struct {
		name    string
		params  Params
		want    Result
		wantErr bool
	}{
		{
			// Acceptance criterion 2: 1000 ₽ subtotal, 800 bonuses on the card,
			// half spent by bonuses → 500 spent, 500 ₽ paid, 25 earned.
			name: "half of subtotal by bonuses",
			params: Params{
				Items:         []SaleLine{{ProductID: 1, Qty: 2, PriceKop: 50000}},
				CustomerID:    id(1),
				BonusBalance:  800,
				BonusPercent:  50,
				CashbackPct:   5,
				SpendLimitPct: 50,
			},
			want: Result{SubtotalKop: 100000, BonusSpent: 500, TotalKop: 50000, BonusEarned: 25},
		},
		{
			name: "no card means no bonuses",
			params: Params{
				Items:         []SaleLine{{ProductID: 1, Qty: 1, PriceKop: 50000}},
				BonusPercent:  50,
				CashbackPct:   5,
				SpendLimitPct: 50,
			},
			want: Result{SubtotalKop: 50000, BonusSpent: 0, TotalKop: 50000, BonusEarned: 0},
		},
		{
			// A balance of 120 points is 120 ₽, below the 250 ₽ cap of the sale.
			name: "balance lower than the cap",
			params: Params{
				Items:         []SaleLine{{ProductID: 1, Qty: 1, PriceKop: 50000}},
				CustomerID:    id(1),
				BonusBalance:  120,
				BonusPercent:  50,
				CashbackPct:   5,
				SpendLimitPct: 50,
			},
			want: Result{SubtotalKop: 50000, BonusSpent: 120, TotalKop: 38000, BonusEarned: 19},
		},
		{
			name: "zero balance with a request",
			params: Params{
				Items:         []SaleLine{{ProductID: 1, Qty: 1, PriceKop: 50000}},
				CustomerID:    id(1),
				BonusBalance:  0,
				BonusPercent:  50,
				CashbackPct:   5,
				SpendLimitPct: 50,
			},
			want: Result{SubtotalKop: 50000, BonusSpent: 0, TotalKop: 50000, BonusEarned: 25},
		},
		{
			// Cashback is computed in kopecks and floored to whole points:
			// 5.00 ₽ paid × 5 % = 0.25 ₽ → 0 points.
			name: "cashback below one point is dropped",
			params: Params{
				Items:         []SaleLine{{ProductID: 1, Qty: 1, PriceKop: 10000}},
				CustomerID:    id(1),
				BonusBalance:  1000,
				BonusPercent:  0,
				CashbackPct:   5,
				SpendLimitPct: 50,
			},
			want: Result{SubtotalKop: 10000, BonusSpent: 0, TotalKop: 10000, BonusEarned: 5},
		},
		{
			// 333 kopecks paid: 5 % is 16 kopecks, floored to 0 points.
			name: "cashback rounds down to whole points",
			params: Params{
				Items:         []SaleLine{{ProductID: 1, Qty: 1, PriceKop: 333}},
				CustomerID:    id(1),
				BonusBalance:  1000,
				BonusPercent:  0,
				CashbackPct:   5,
				SpendLimitPct: 50,
			},
			want: Result{SubtotalKop: 333, BonusSpent: 0, TotalKop: 333, BonusEarned: 0},
		},
		{
			// 99 kopecks, 50 % → 49 kopecks, floored to 0 points spent, so the
			// customer keeps the kopecks rather than the shop losing them.
			name: "bonus spend below one point is dropped",
			params: Params{
				Items:         []SaleLine{{ProductID: 1, Qty: 1, PriceKop: 99}},
				CustomerID:    id(1),
				BonusBalance:  1000,
				BonusPercent:  50,
				CashbackPct:   5,
				SpendLimitPct: 50,
			},
			want: Result{SubtotalKop: 99, BonusSpent: 0, TotalKop: 99, BonusEarned: 0},
		},
		{
			// 250 kopecks, 50 % → 125 kopecks → 1 point spent, 150 kopecks left.
			name: "bonus spend floors to whole points",
			params: Params{
				Items:         []SaleLine{{ProductID: 1, Qty: 1, PriceKop: 250}},
				CustomerID:    id(1),
				BonusBalance:  1000,
				BonusPercent:  50,
				CashbackPct:   5,
				SpendLimitPct: 50,
			},
			want: Result{SubtotalKop: 250, BonusSpent: 1, TotalKop: 150, BonusEarned: 0},
		},
		{
			name: "several lines",
			params: Params{
				Items: []SaleLine{
					{ProductID: 3, Qty: 2, PriceKop: 15000},
					{ProductID: 1, Qty: 1, PriceKop: 99900},
				},
				CustomerID:    id(2),
				BonusBalance:  100000,
				BonusPercent:  50,
				CashbackPct:   5,
				SpendLimitPct: 50,
			},
			// 1299 ₽ × 50 % = 649,50 ₽ → 649 whole points, 650 ₽ left in cash.
			want: Result{SubtotalKop: 129900, BonusSpent: 649, TotalKop: 65000, BonusEarned: 32},
		},
		{
			name: "empty cart",
			params: Params{
				Items:         nil,
				CashbackPct:   5,
				SpendLimitPct: 50,
			},
			want:    Result{},
			wantErr: true,
		},
		{
			name: "percent above the limit",
			params: Params{
				Items:         []SaleLine{{ProductID: 1, Qty: 1, PriceKop: 10000}},
				CustomerID:    id(1),
				BonusBalance:  1000,
				BonusPercent:  75,
				CashbackPct:   5,
				SpendLimitPct: 50,
			},
			wantErr: true,
		},
		{
			name: "negative percent",
			params: Params{
				Items:         []SaleLine{{ProductID: 1, Qty: 1, PriceKop: 10000}},
				CustomerID:    id(1),
				BonusBalance:  1000,
				BonusPercent:  -1,
				CashbackPct:   5,
				SpendLimitPct: 50,
			},
			wantErr: true,
		},
		{
			name:    "zero quantity",
			params:  Params{Items: []SaleLine{{ProductID: 1, Qty: 0, PriceKop: 10000}}, SpendLimitPct: 50, CashbackPct: 5},
			wantErr: true,
		},
		{
			name:    "negative price",
			params:  Params{Items: []SaleLine{{ProductID: 1, Qty: 1, PriceKop: -1}}, SpendLimitPct: 50, CashbackPct: 5},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Calculate(tt.params)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestCalculateNeverGoesNegative guards the invariant the DB checks also enforce.
func TestCalculateNeverGoesNegative(t *testing.T) {
	for _, balance := range []int{0, 1, 7, 99} {
		res, err := Calculate(Params{
			Items:         []SaleLine{{ProductID: 1, Qty: 1, PriceKop: 100}},
			CustomerID:    id(1),
			BonusBalance:  balance,
			BonusPercent:  50,
			CashbackPct:   5,
			SpendLimitPct: 50,
		})
		if err != nil {
			t.Fatalf("balance %d: %v", balance, err)
		}
		if res.BonusSpent < 0 || res.TotalKop < 0 || res.BonusEarned < 0 {
			t.Fatalf("balance %d produced a negative amount: %+v", balance, res)
		}
		if res.TotalKop > res.SubtotalKop {
			t.Fatalf("balance %d: total %d exceeds subtotal %d", balance, res.TotalKop, res.SubtotalKop)
		}
		if int64(res.BonusSpent) > int64(balance) {
			t.Fatalf("balance %d: spent %d exceeds the balance", balance, res.BonusSpent)
		}
	}
}

// TestCashbackUsesCashOnlyAmount: the cashback base is what was paid in cash, so a
// bonus-heavy sale earns less than the same sale without bonuses.
func TestCashbackUsesCashOnlyAmount(t *testing.T) {
	lines := []SaleLine{{ProductID: 1, Qty: 1, PriceKop: 100000}}

	withoutBonuses, err := Calculate(Params{
		Items: lines, CustomerID: id(1), BonusBalance: 1000, BonusPercent: 0,
		CashbackPct: 5, SpendLimitPct: 50,
	})
	if err != nil {
		t.Fatalf("without bonuses: %v", err)
	}
	withBonuses, err := Calculate(Params{
		Items: lines, CustomerID: id(1), BonusBalance: 1000, BonusPercent: 50,
		CashbackPct: 5, SpendLimitPct: 50,
	})
	if err != nil {
		t.Fatalf("with bonuses: %v", err)
	}

	if withoutBonuses.BonusEarned != 50 {
		t.Fatalf("1000 ₽ paid should earn 50 bonuses, got %d", withoutBonuses.BonusEarned)
	}
	if withBonuses.BonusEarned != 25 {
		t.Fatalf("500 ₽ paid should earn 25 bonuses, got %d", withBonuses.BonusEarned)
	}
}