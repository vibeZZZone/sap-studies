package bonus

import "fmt"

// Units. Subtotal and total are kopecks. Bonus amounts are bonus points, where one
// point is worth one ruble, which is what the customers table stores.
const (
	kopPerBonus = 100
)

type SaleLine struct {
	ProductID int   `json:"productId"`
	Qty       int   `json:"qty"`
	PriceKop  int64 `json:"priceKop"`
}

// Params are the inputs of a sale calculation. Product lines arrive with the price
// already locked inside the transaction, so the function stays pure.
type Params struct {
	Items []SaleLine
	// CustomerID is nil for a sale without a card, in which case no bonuses apply.
	CustomerID *int
	// BonusBalance is the bonus balance of the customer in bonus points.
	BonusBalance int
	// BonusPercent is what the client asked to spend, 0..SpendLimitPct.
	BonusPercent int
	CashbackPct  int
	SpendLimitPct int
}

// Result carries SubtotalKop and TotalKop in kopecks, and the bonus amounts in bonus
// points. Mixing the two units in one integer is exactly the mistake this type is
// meant to prevent, so the field names say which unit each value uses.
type Result struct {
	SubtotalKop int64
	BonusSpent  int64
	TotalKop    int64
	BonusEarned int64
}

var ErrEmptyCart = fmt.Errorf("cart is empty")

// Calculate applies the bonus program rules to a set of lines.
//
//	subtotal      = sum(priceKop * qty)
//	bonusSpent    = min(floor(subtotal * bonusPercent / 100), bonusBalance)
//	total         = subtotal - bonusSpent * 100
//	bonusEarned   = floor(total * cashbackPct / 100 / 100)
//
// The first two steps work in kopecks, so a bonus point is worth exactly 100 kopecks.
// The cashback is computed on the amount the customer actually paid in cash, after
// the bonus deduction. Every division floors: the shop never over-promises a bonus.
// Without a customer card nothing is spent or earned, whatever BonusPercent says.
func Calculate(p Params) (Result, error) {
	if len(p.Items) == 0 {
		return Result{}, ErrEmptyCart
	}
	if p.BonusPercent < 0 || p.BonusPercent > p.SpendLimitPct {
		return Result{}, fmt.Errorf("bonus percent %d is outside [0,%d]", p.BonusPercent, p.SpendLimitPct)
	}

	var subtotal int64
	for _, item := range p.Items {
		if item.Qty <= 0 {
			return Result{}, fmt.Errorf("quantity for product %d must be positive", item.ProductID)
		}
		if item.PriceKop < 0 {
			return Result{}, fmt.Errorf("price for product %d must not be negative", item.ProductID)
		}
		subtotal += item.PriceKop * int64(item.Qty)
	}

	res := Result{SubtotalKop: subtotal, TotalKop: subtotal}
	if p.CustomerID == nil {
		return res, nil
	}

	if p.BonusPercent > 0 {
		spendKop := subtotal * int64(p.BonusPercent) / 100
		maxSpendKop := int64(p.BonusBalance) * kopPerBonus
		if spendKop > maxSpendKop {
			spendKop = maxSpendKop
		}
		if spendKop < 0 {
			spendKop = 0
		}
		// A partial point cannot be deducted: rounding here keeps the total equal to
		// subtotal minus exactly the points taken off the card.
		spendKop -= spendKop % kopPerBonus
		res.BonusSpent = spendKop / kopPerBonus
		res.TotalKop = subtotal - spendKop
	}
	if p.CashbackPct > 0 {
		res.BonusEarned = res.TotalKop * int64(p.CashbackPct) / 100 / kopPerBonus
	}
	return res, nil
}