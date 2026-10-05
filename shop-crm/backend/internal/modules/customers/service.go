package customers

import (
	"context"
	"strings"

	apperrors "github.com/sap-studies/shop-crm/backend/internal/lib/errors"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, search string) ([]Customer, error) {
	return s.repo.List(ctx, strings.TrimSpace(search))
}

type CreateInput struct {
	Name  string
	Phone string
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Customer, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Customer{}, apperrors.Validation("Customer name is required")
	}
	if err := validateName(name); err != nil {
		return Customer{}, apperrors.Validation(err.Error())
	}

	phone := normalizePhone(in.Phone)
	if phone == "" {
		return Customer{}, apperrors.Validation("Phone number is required")
	}
	if !validPhone(phone) {
		return Customer{}, apperrors.Validation("Phone number format is not valid")
	}

	c, err := s.repo.Create(ctx, name, phone)
	if isUniqueViolation(err, "phone") {
		return Customer{}, apperrors.Conflict(apperrors.CodePhoneDuplicate, "A customer with this phone already has a card")
	}
	return c, err
}

// validPhone accepts digits plus the usual separators and requires 6 to 32
// characters overall.
func validPhone(phone string) bool {
	if len(phone) < 6 || len(phone) > 32 {
		return false
	}
	digits := 0
	for _, r := range phone {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == '+' || r == ' ' || r == '-' || r == '(' || r == ')':
		default:
			return false
		}
	}
	return digits >= 6
}