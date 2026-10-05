package products

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Product struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	PriceKop  int64   `json:"priceKop"`
	Price     float64 `json:"price"`
	Stock     int     `json:"stock"`
	Archived  bool    `json:"archived"`
	LowStock  bool    `json:"lowStock"`
	CreatedAt string  `json:"createdAt"`
}

type CreateParams struct {
	Name       string
	PriceKop   int64
	InitialQty int
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

var ErrNotFound = errors.New("product not found")

// List returns non-archived products ordered the way the till shows them.
func (r *Repository) List(ctx context.Context) ([]Product, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, price_kop, stock, archived, created_at
		FROM products
		WHERE NOT archived
		ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	products := make([]Product, 0)
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *Repository) Create(ctx context.Context, in CreateParams) (Product, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO products (name, price_kop, stock)
		VALUES ($1, $2, $3)
		RETURNING id, name, price_kop, stock, archived, created_at`,
		in.Name, in.PriceKop, in.InitialQty)
	p, err := scan(row)
	if err != nil {
		return Product{}, fmt.Errorf("create product: %w", err)
	}
	return p, nil
}

// Restock adds qty units of stock. The UPDATE ... WHERE stock + $2 >= 0 guard keeps
// the non-negative invariant even if a caller ever passes a negative value.
func (r *Repository) Restock(ctx context.Context, id, qty int) (Product, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE products
		SET stock = stock + $2
		WHERE id = $1 AND NOT archived AND $2 > 0
		RETURNING id, name, price_kop, stock, archived, created_at`, id, qty)
	p, err := scan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	if err != nil {
		return Product{}, fmt.Errorf("restock product: %w", err)
	}
	return p, nil
}

// Archive performs the soft delete: the row stays so historical receipts keep
// pointing at a real product.
func (r *Repository) Archive(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, `UPDATE products SET archived = true WHERE id = $1 AND NOT archived`, id)
	if err != nil {
		return fmt.Errorf("archive product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scan(row rowScanner) (Product, error) {
	var p Product
	var createdAt any
	err := row.Scan(&p.ID, &p.Name, &p.PriceKop, &p.Stock, &p.Archived, &createdAt)
	if err != nil {
		return Product{}, err
	}
	p.Price = float64(p.PriceKop) / 100
	p.LowStock = p.Stock <= 5
	if t, ok := createdAt.(interface{ Format(string) string }); ok {
		p.CreatedAt = t.Format("2006-01-02T15:04:05Z07:00")
	}
	return p, nil
}