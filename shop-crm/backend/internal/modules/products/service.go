package products

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	apperrors "github.com/sap-studies/shop-crm/backend/internal/lib/errors"
	"github.com/sap-studies/shop-crm/backend/internal/lib/money"
)

const (
	maxNameLen = 200
	maxQty     = 100000
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) ([]Product, error) {
	return s.repo.List(ctx)
}

// CreateInput carries the price as a ruble string so the client never deals with
// float precision; the server parses it into kopecks.
type CreateInput struct {
	Name        string
	PriceRubles string
	InitialQty  int
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Product, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Product{}, apperrors.Validation("Product name is required")
	}
	if utf8.RuneCountInString(name) > maxNameLen {
		return Product{}, apperrors.Validation("Product name is too long")
	}

	priceKop, err := money.RubToKop(strings.TrimSpace(in.PriceRubles))
	if err != nil || priceKop <= 0 {
		return Product{}, apperrors.Validation("Price must be a positive amount with at most two decimals")
	}
	if in.InitialQty < 0 || in.InitialQty > maxQty {
		return Product{}, apperrors.Validation("Initial quantity must be zero or greater")
	}

	return s.repo.Create(ctx, CreateParams{Name: name, PriceKop: priceKop, InitialQty: in.InitialQty})
}

func (s *Service) Restock(ctx context.Context, id, qty int) (Product, error) {
	if qty <= 0 {
		return Product{}, apperrors.Validation("Quantity must be positive")
	}
	if qty > maxQty {
		return Product{}, apperrors.Validation("Quantity is too large")
	}
	p, err := s.repo.Restock(ctx, id, qty)
	if errors.Is(err, ErrNotFound) {
		return Product{}, apperrors.NotFound(apperrors.CodeProductNotFound, "Product not found")
	}
	return p, err
}

func (s *Service) Archive(ctx context.Context, id int) error {
	err := s.repo.Archive(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return apperrors.NotFound(apperrors.CodeProductNotFound, "Product not found")
	}
	return err
}

