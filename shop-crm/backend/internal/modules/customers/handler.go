package customers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

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
	r.Get("/customers", h.list)
	r.Post("/customers", h.create)
}

type createRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	customers, err := h.svc.List(r.Context(), r.URL.Query().Get("search"))
	if err != nil {
		apperrors.Write(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, customers)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := decodeJSON(r, &req); err != nil {
		apperrors.Write(w, h.log, err)
		return
	}
	customer, err := h.svc.Create(r.Context(), CreateInput{Name: req.Name, Phone: req.Phone})
	if err != nil {
		apperrors.Write(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusCreated, customer)
}

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

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}