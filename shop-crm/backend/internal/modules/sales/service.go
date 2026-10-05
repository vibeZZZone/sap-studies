package sales

import (
	"context"
	"errors"
	"sort"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sap-studies/shop-crm/backend/internal/lib/bonus"
	apperrors "github.com/sap-studies/shop-crm/backend/internal/lib/errors"
)

const maxQtyPerLine = 100000

type CreateInput struct {
	CustomerID  *int
	BonusPercent int
	Items       []ItemInput
}

type ItemInput struct {
	ProductID int `json:"productId"`
	Qty       int `json:"qty"`
}

type Service struct {
	repo      *Repository
	pool      *pgxpool.Pool
	cashbackPct int
	spendLimitPct int
}

func NewService(repo *Repository, pool *pgxpool.Pool, cashbackPct, spendLimitPct int) *Service {
	return &Service{repo: repo, pool: pool, cashbackPct: cashbackPct, spendLimitPct: spendLimitPct}
}

func (s *Service) List(ctx context.Context, page int) ([]Sale, int, error) {
	return s.repo.List(ctx, page)
}

// Create runs the whole sale in one transaction:
//
//  1. lock the product rows and the customer row FOR UPDATE,
//  2. check stock and calculate the totals from the locked prices,
//  3. insert the sale and its items,
//  4. decrement stock,
//  5. update the bonus balance and total spent.
//
// Any failure rolls everything back, so stock and bonuses move together or not at
// all. The client never sends amounts: the server is the only source of truth.
func (s *Service) Create(ctx context.Context, in CreateInput) (Sale, error) {
	if err := s.validateInput(in); err != nil {
		return Sale{}, err
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return Sale{}, apperrors.Wrap(err, 500, apperrors.CodeInternal, "Could not start the sale")
	}
	// Rollback is a no-op once the transaction has been committed.
	defer func() { _ = tx.Rollback(ctx) }()

	ids := make([]int, 0, len(in.Items))
	for _, item := range in.Items {
		ids = append(ids, item.ProductID)
	}
	sort.Ints(ids)

	locked, err := s.repo.lockProducts(ctx, tx, ids)
	if err != nil {
		return Sale{}, apperrors.Wrap(err, 500, apperrors.CodeInternal, "Could not read products")
	}
	for _, item := range in.Items {
		p, ok := locked[item.ProductID]
		if !ok {
			return Sale{}, apperrors.WithParam(
				apperrors.NotFound(apperrors.CodeProductNotFound, "Product not found"), "productId", strconv.Itoa(item.ProductID))
		}
		if item.Qty > p.Stock {
			return Sale{}, apperrors.WithParam(
				apperrors.Conflict(apperrors.CodeInsufficientStock,
					"Not enough stock for this product"),
				"productId", strconv.Itoa(item.ProductID))
		}
	}

	var customer lockedCustomer
	if in.CustomerID != nil {
		customer, err = s.repo.lockCustomer(ctx, tx, *in.CustomerID)
		if errors.Is(err, ErrNotFound) {
			return Sale{}, apperrors.NotFound(apperrors.CodeCustomerNotFound, "Customer not found")
		}
		if err != nil {
			return Sale{}, apperrors.Wrap(err, 500, apperrors.CodeInternal, "Could not read customer")
		}
	}

	lines := make([]bonus.SaleLine, 0, len(in.Items))
	for _, item := range in.Items {
		p := locked[item.ProductID]
		lines = append(lines, bonus.SaleLine{
			ProductID: p.ID,
			Qty:       item.Qty,
			PriceKop:  p.PriceKop,
		})
	}

	balance := 0
	if in.CustomerID != nil {
		balance = customer.BonusBalance
	}
	calc, err := bonus.Calculate(bonus.Params{
		Items:           lines,
		CustomerID:      in.CustomerID,
		BonusBalance:    balance,
		BonusPercent:    in.BonusPercent,
		CashbackPct:     s.cashbackPct,
		SpendLimitPct:   s.spendLimitPct,
	})
	if err != nil {
		if errors.Is(err, bonus.ErrEmptyCart) {
			return Sale{}, apperrors.Validation("Cart is empty")
		}
		return Sale{}, apperrors.Validation(err.Error())
	}

	saleID, err := s.repo.insertSale(ctx, tx, in.CustomerID,
		calc.SubtotalKop, calc.BonusSpent, calc.TotalKop, calc.BonusEarned)
	if err != nil {
		return Sale{}, apperrors.Wrap(err, 500, apperrors.CodeInternal, "Could not save the sale")
	}

	items := make([]Item, 0, len(lines))
	for _, line := range lines {
		p := locked[line.ProductID]
		item := Item{
			ProductID: line.ProductID,
			Name:      p.Name,
			PriceKop:  p.PriceKop,
			Qty:       line.Qty,
			SumKop:    line.PriceKop * int64(line.Qty),
		}
		affected, err := s.repo.decrementStock(ctx, tx, line.ProductID, line.Qty)
		if err != nil {
			return Sale{}, apperrors.Wrap(err, 500, apperrors.CodeInternal, "Could not update stock")
		}
		if affected == 0 {
			return Sale{}, apperrors.WithParam(
				apperrors.Conflict(apperrors.CodeInsufficientStock, "Not enough stock for this product"),
				"productId", strconv.Itoa(line.ProductID))
		}
		items = append(items, item)
	}

	if err := s.repo.insertItems(ctx, tx, saleID, items); err != nil {
		return Sale{}, apperrors.Wrap(err, 500, apperrors.CodeInternal, "Could not save the sale items")
	}

	if in.CustomerID != nil {
		if err := s.repo.updateCustomerBonus(ctx, tx, *in.CustomerID, int64(calc.BonusEarned)-calc.BonusSpent, calc.TotalKop); err != nil {
			return Sale{}, apperrors.Wrap(err, 500, apperrors.CodeInternal, "Could not update the bonus balance")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Sale{}, apperrors.Wrap(err, 500, apperrors.CodeInternal, "Could not complete the sale")
	}

	sale, err := s.repo.Get(ctx, saleID)
	if err != nil {
		return Sale{}, apperrors.Wrap(err, 500, apperrors.CodeInternal, "Could not load the saved sale")
	}
	return sale, nil
}

// validateInput rejects a malformed request before a transaction is opened.
func (s *Service) validateInput(in CreateInput) error {
	if len(in.Items) == 0 {
		return apperrors.New(400, apperrors.CodeEmptyCart, "Cart is empty")
	}
	if in.CustomerID != nil && *in.CustomerID <= 0 {
		return apperrors.Validation("Customer id must be a positive integer")
	}
	if in.BonusPercent < 0 || in.BonusPercent > s.spendLimitPct {
		return apperrors.Validation("Bonus percent must be between 0 and " + strconv.Itoa(s.spendLimitPct))
	}

	seen := make(map[int]struct{}, len(in.Items))
	for _, item := range in.Items {
		if item.ProductID <= 0 {
			return apperrors.Validation("Product id must be a positive integer")
		}
		if item.Qty <= 0 || item.Qty > maxQtyPerLine {
			return apperrors.Validation("Quantity must be a positive number")
		}
		if _, dup := seen[item.ProductID]; dup {
			return apperrors.Validation("The cart contains the same product twice")
		}
		seen[item.ProductID] = struct{}{}
	}
	return nil
}