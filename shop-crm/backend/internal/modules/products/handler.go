package products

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	apperrors "github.com/sap-studies/shop-crm/backend/internal/lib/errors"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
	log *slog.Logger
}

func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Mount(r chi.Router) {
	r.Route("/products", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Post("/{id}/restock", h.restock)
		r.Delete("/{id}", h.archive)
	})
}

type createRequest struct {
	Name       string `json:"name"`
	Price      string `json:"price"`
	InitialQty int    `json:"initialQty"`
}

type restockRequest struct {
	Qty int `json:"qty"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	products, err := h.svc.List(r.Context())
	if err != nil {
		apperrors.Write(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, products)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := decodeJSON(r, &req); err != nil {
		apperrors.Write(w, h.log, err)
		return
	}
	product, err := h.svc.Create(r.Context(), CreateInput{
		Name:        req.Name,
		PriceRubles: req.Price,
		InitialQty:  req.InitialQty,
	})
	if err != nil {
		apperrors.Write(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusCreated, product)
}

func (h *Handler) restock(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, h.log, r)
	if !ok {
		return
	}
	var req restockRequest
	if err := decodeJSON(r, &req); err != nil {
		apperrors.Write(w, h.log, err)
		return
	}
	product, err := h.svc.Restock(r.Context(), id, req.Qty)
	if err != nil {
		apperrors.Write(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, product)
}

func (h *Handler) archive(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, h.log, r)
	if !ok {
		return
	}
	if err := h.svc.Archive(r.Context(), id); err != nil {
		apperrors.Write(w, h.log, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// decodeJSON rejects unknown fields so a typo in the client payload surfaces as a
// 400 instead of being silently ignored.
func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return apperrors.Validation("Request body is required")
		}
		return apperrors.Validation("Request body is not valid JSON")
	}
	return nil
}

func pathID(w http.ResponseWriter, log *slog.Logger, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		apperrors.Write(w, log, apperrors.Validation("Id must be a positive integer"))
		return 0, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}