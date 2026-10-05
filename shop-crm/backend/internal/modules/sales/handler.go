package sales

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/sap-studies/shop-crm/backend/internal/lib/errors"
)

type Handler struct {
	svc *Service
	log *slog.Logger
}

func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Mount(r chi.Router) {
	r.Get("/sales", h.list)
	r.Post("/sales", h.create)
}

type createRequest struct {
	CustomerID   *int         `json:"customerId"`
	BonusPercent int          `json:"bonusPercent"`
	Items        []ItemInput  `json:"items"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			apperrors.Write(w, h.log, apperrors.Validation("Page must be a positive integer"))
			return
		}
		page = parsed
	}

	sales, total, err := h.svc.List(r.Context(), page)
	if err != nil {
		apperrors.Write(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"sales": sales,
		"page":  page,
		"pageSize": 20,
		"total": total,
		"pages":  (total + 19) / 20,
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			apperrors.Write(w, h.log, apperrors.Validation("Request body is required"))
			return
		}
		apperrors.Write(w, h.log, apperrors.Validation("Request body is not valid JSON"))
		return
	}

	sale, err := h.svc.Create(r.Context(), CreateInput{
		CustomerID:   req.CustomerID,
		BonusPercent: req.BonusPercent,
		Items:        req.Items,
	})
	if err != nil {
		apperrors.Write(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusCreated, sale)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}