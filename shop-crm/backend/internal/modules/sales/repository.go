package sales

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sap-studies/shop-crm/backend/internal/lib/money"
)

var ErrNotFound = errors.New("sale not found")

type Item struct {
	ProductID int    `json:"productId"`
	Name      string `json:"name"`
	PriceKop  int64  `json:"priceKop"`
	Price     string `json:"price"`
	Qty       int    `json:"qty"`
	SumKop    int64  `json:"sumKop"`
	Sum       string `json:"sum"`
}

type CustomerRef struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	CardNumber string `json:"cardNumber"`
}

type Sale struct {
	ID           int         `json:"id"`
	CustomerID   *int        `json:"customerId"`
	Customer     *CustomerRef `json:"customer"`
	SubtotalKop  int64       `json:"subtotalKop"`
	Subtotal     string      `json:"subtotal"`
	BonusSpent   int         `json:"bonusSpent"`
	TotalKop     int64       `json:"totalKop"`
	Total        string      `json:"total"`
	BonusEarned  int         `json:"bonusEarned"`
	CreatedAt    string      `json:"createdAt"`
	Items        []Item      `json:"items"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// lockedProduct is a product row locked FOR UPDATE inside the sale transaction.
type lockedProduct struct {
	ID      int
	Name    string
	PriceKop int64
	Stock   int
}

// lockProducts takes row locks on the given products in a stable id order. Sorting
// first is what stops two concurrent sales for the same pair of products from
// deadlocking against each other.
func (r *Repository) lockProducts(ctx context.Context, tx pgx.Tx, ids []int) (map[int]lockedProduct, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, name, price_kop, stock
		FROM products
		WHERE id = ANY($1) AND NOT archived
		ORDER BY id
		FOR UPDATE`, ids)
	if err != nil {
		return nil, fmt.Errorf("lock products: %w", err)
	}
	defer rows.Close()

	locked := make(map[int]lockedProduct, len(ids))
	for rows.Next() {
		var p lockedProduct
		if err := rows.Scan(&p.ID, &p.Name, &p.PriceKop, &p.Stock); err != nil {
			return nil, fmt.Errorf("scan locked product: %w", err)
		}
		locked[p.ID] = p
	}
	return locked, rows.Err()
}

type lockedCustomer struct {
	ID           int
	Name         string
	CardNumber   string
	BonusBalance int
}

func (r *Repository) lockCustomer(ctx context.Context, tx pgx.Tx, id int) (lockedCustomer, error) {
	var c lockedCustomer
	err := tx.QueryRow(ctx, `
		SELECT id, name, card_number, bonus_balance
		FROM customers
		WHERE id = $1
		FOR UPDATE`, id).Scan(&c.ID, &c.Name, &c.CardNumber, &c.BonusBalance)
	if errors.Is(err, pgx.ErrNoRows) {
		return lockedCustomer{}, ErrNotFound
	}
	if err != nil {
		return lockedCustomer{}, fmt.Errorf("lock customer: %w", err)
	}
	return c, nil
}

func (r *Repository) insertSale(ctx context.Context, tx pgx.Tx, customerID *int, subtotal, bonusSpent, total, bonusEarned int64) (int, error) {
	var id int
	err := tx.QueryRow(ctx, `
		INSERT INTO sales (customer_id, subtotal_kop, bonus_spent, total_kop, bonus_earned)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`, customerID, subtotal, bonusSpent, total, bonusEarned).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert sale: %w", err)
	}
	return id, nil
}

// insertItems snapshots name and price so later price changes never rewrite history.
func (r *Repository) insertItems(ctx context.Context, tx pgx.Tx, saleID int, items []Item) error {
	for _, item := range items {
		_, err := tx.Exec(ctx, `
			INSERT INTO sale_items (sale_id, product_id, name_snapshot, price_kop, qty)
			VALUES ($1, $2, $3, $4, $5)`,
			saleID, item.ProductID, item.Name, item.PriceKop, item.Qty)
		if err != nil {
			return fmt.Errorf("insert sale item: %w", err)
		}
	}
	return nil
}

// decrementStock subtracts the sold quantity. The WHERE guard makes the update a
// no-op if the quantity ever went above the locked stock, and the caller checks the
// affected rows.
func (r *Repository) decrementStock(ctx context.Context, tx pgx.Tx, productID int, qty int) (int64, error) {
	tag, err := tx.Exec(ctx, `
		UPDATE products
		SET stock = stock - $2
		WHERE id = $1 AND stock >= $2`, productID, qty)
	if err != nil {
		return 0, fmt.Errorf("decrement stock: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *Repository) updateCustomerBonus(ctx context.Context, tx pgx.Tx, customerID int, delta, totalPaid int64) error {
	_, err := tx.Exec(ctx, `
		UPDATE customers
		SET bonus_balance = bonus_balance + $2,
		    total_spent_kop = total_spent_kop + $3
		WHERE id = $1`, customerID, delta, totalPaid)
	if err != nil {
		return fmt.Errorf("update customer bonus: %w", err)
	}
	return nil
}

const salesPageSize = 20

// List returns a page of sales with their items, newest first.
func (r *Repository) List(ctx context.Context, page int) ([]Sale, int, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * salesPageSize

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM sales`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count sales: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.customer_id, s.subtotal_kop, s.bonus_spent, s.total_kop,
		       s.bonus_earned, s.created_at, c.name, c.card_number
		FROM sales s
		LEFT JOIN customers c ON c.id = s.customer_id
		ORDER BY s.created_at DESC, s.id DESC
		LIMIT $1 OFFSET $2`, salesPageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list sales: %w", err)
	}
	defer rows.Close()

	sales := make([]Sale, 0, salesPageSize)
	for rows.Next() {
		var s Sale
		var createdAt time.Time
		var custName, cardNumber *string
		if err := rows.Scan(&s.ID, &s.CustomerID, &s.SubtotalKop, &s.BonusSpent, &s.TotalKop,
			&s.BonusEarned, &createdAt, &custName, &cardNumber); err != nil {
			return nil, 0, fmt.Errorf("scan sale: %w", err)
		}
		s.Subtotal = money.FormatKop(s.SubtotalKop)
		s.Total = money.FormatKop(s.TotalKop)
		s.CreatedAt = createdAt.Format(time.RFC3339)
		if s.CustomerID != nil && custName != nil && cardNumber != nil {
			s.Customer = &CustomerRef{ID: *s.CustomerID, Name: *custName, CardNumber: *cardNumber}
		}
		s.Items = []Item{}
		sales = append(sales, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if len(sales) == 0 {
		return sales, total, nil
	}

	// The index is built after the loop: pointers into a slice that is still growing
	// would be invalidated by a reallocation.
	indexByID := make(map[int]int, len(sales))
	for i := range sales {
		indexByID[sales[i].ID] = i
	}
	ids := make([]int, 0, len(sales))
	for id := range indexByID {
		ids = append(ids, id)
	}
	itemRows, err := r.pool.Query(ctx, `
		SELECT sale_id, product_id, name_snapshot, price_kop, qty
		FROM sale_items
		WHERE sale_id = ANY($1)
		ORDER BY id`, ids)
	if err != nil {
		return nil, 0, fmt.Errorf("list sale items: %w", err)
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var saleID int
		var item Item
		if err := itemRows.Scan(&saleID, &item.ProductID, &item.Name, &item.PriceKop, &item.Qty); err != nil {
			return nil, 0, fmt.Errorf("scan sale item: %w", err)
		}
		item.Price = money.FormatKop(item.PriceKop)
		item.SumKop = item.PriceKop * int64(item.Qty)
		item.Sum = money.FormatKop(item.SumKop)
		if i, ok := indexByID[saleID]; ok {
			sales[i].Items = append(sales[i].Items, item)
		}
	}
	return sales, total, itemRows.Err()
}

func (r *Repository) Get(ctx context.Context, id int) (Sale, error) {
	sales, _, err := r.listOne(ctx, id)
	if err != nil {
		return Sale{}, err
	}
	if len(sales) == 0 {
		return Sale{}, ErrNotFound
	}
	return sales[0], nil
}

func (r *Repository) listOne(ctx context.Context, id int) ([]Sale, int, error) {
	var s Sale
	var createdAt time.Time
	var custName, cardNumber *string
	err := r.pool.QueryRow(ctx, `
		SELECT s.id, s.customer_id, s.subtotal_kop, s.bonus_spent, s.total_kop,
		       s.bonus_earned, s.created_at, c.name, c.card_number
		FROM sales s
		LEFT JOIN customers c ON c.id = s.customer_id
		WHERE s.id = $1`, id).
		Scan(&s.ID, &s.CustomerID, &s.SubtotalKop, &s.BonusSpent, &s.TotalKop,
			&s.BonusEarned, &createdAt, &custName, &cardNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("get sale: %w", err)
	}
	s.Subtotal = money.FormatKop(s.SubtotalKop)
	s.Total = money.FormatKop(s.TotalKop)
	s.CreatedAt = createdAt.Format(time.RFC3339)
	if s.CustomerID != nil && custName != nil && cardNumber != nil {
		s.Customer = &CustomerRef{ID: *s.CustomerID, Name: *custName, CardNumber: *cardNumber}
	}

	rows, err := r.pool.Query(ctx, `
		SELECT product_id, name_snapshot, price_kop, qty
		FROM sale_items
		WHERE sale_id = $1
		ORDER BY id`, id)
	if err != nil {
		return nil, 0, fmt.Errorf("get sale items: %w", err)
	}
	defer rows.Close()

	s.Items = []Item{}
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ProductID, &item.Name, &item.PriceKop, &item.Qty); err != nil {
			return nil, 0, err
		}
		item.Price = money.FormatKop(item.PriceKop)
		item.SumKop = item.PriceKop * int64(item.Qty)
		item.Sum = money.FormatKop(item.SumKop)
		s.Items = append(s.Items, item)
	}
	return []Sale{s}, 1, rows.Err()
}