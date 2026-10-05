package customers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sap-studies/shop-crm/backend/internal/lib/money"
)

// CardPrefix is the fixed part of every issued card number: 5000-0001. The running
// number comes from the card_number_seq sequence in the database.
const CardPrefix = "5000"

var ErrNotFound = errors.New("customer not found")

// isUniqueViolation reports whether err is a Postgres 23505 on the given column.
func isUniqueViolation(err error, column string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return column == "" || pgErr.ConstraintName == "customers_"+column+"_key"
	}
	return false
}

type Customer struct {
	ID            int    `json:"id"`
	CardNumber    string `json:"cardNumber"`
	Name          string `json:"name"`
	Phone         string `json:"phone"`
	BonusBalance  int    `json:"bonusBalance"`
	TotalSpentKop int64  `json:"totalSpentKop"`
	TotalSpent    string `json:"totalSpent"`
	CreatedAt     string `json:"createdAt"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const columns = `id, card_number, name, phone, bonus_balance, total_spent_kop, created_at`

// List returns customers matching search. An empty search lists everyone. Filtering
// happens server-side across name, phone and card number.
func (r *Repository) List(ctx context.Context, search string) ([]Customer, error) {
	pattern := "%" + strings.ToLower(search) + "%"
	rows, err := r.pool.Query(ctx, `
		SELECT `+columns+`
		FROM customers
		WHERE $1 = ''
		   OR lower(name)        LIKE $2
		   OR phone              LIKE $2
		   OR lower(card_number) LIKE $2
		ORDER BY id`, search, pattern)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()

	customers := make([]Customer, 0)
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, err
		}
		customers = append(customers, c)
	}
	return customers, rows.Err()
}

// Create issues the next card number in the 5000-0001 sequence and inserts the
// customer. The sequence lives in the database, so two concurrent requests never
// receive the same card.
func (r *Repository) Create(ctx context.Context, name, phone string) (Customer, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO customers (card_number, name, phone, bonus_balance)
		VALUES (
			'` + CardPrefix + `' || '-' || lpad(nextval('card_number_seq')::text, 4, '0'),
			$1,
			$2,
			0
		)
		RETURNING `+columns, name, phone)

	c, err := scan(row)
	if err != nil {
		return Customer{}, err
	}
	return c, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scan(row rowScanner) (Customer, error) {
	var c Customer
	var createdAt any
	if err := row.Scan(&c.ID, &c.CardNumber, &c.Name, &c.Phone, &c.BonusBalance, &c.TotalSpentKop, &createdAt); err != nil {
		return Customer{}, err
	}
	c.TotalSpent = money.FormatKop(c.TotalSpentKop)
	if t, ok := createdAt.(interface{ Format(string) string }); ok {
		c.CreatedAt = t.Format("2006-01-02T15:04:05Z07:00")
	}
	return c, nil
}

// normalizePhone only trims. Anything else is left in place so that validPhone can
// reject a phone with letters instead of silently turning it into a valid one.
func normalizePhone(raw string) string {
	return strings.TrimSpace(raw)
}

func validateName(name string) error {
	if utf8.RuneCountInString(name) > 200 {
		return errors.New("name is too long")
	}
	return nil
}