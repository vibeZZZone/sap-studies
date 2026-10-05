package app

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sap-studies/shop-crm/backend/internal/config"
	"github.com/sap-studies/shop-crm/backend/internal/lib/money"
	"github.com/sap-studies/shop-crm/backend/internal/modules/customers"
	"github.com/sap-studies/shop-crm/backend/internal/modules/products"
	"github.com/sap-studies/shop-crm/backend/internal/modules/sales"
)

// Build wires the HTTP surface. Authentication is out of scope for the MVP, but the
// router is the only place that knows about it, so adding a role check later is a
// middleware rather than a rewrite.
func Build(pool *pgxpool.Pool, cfg config.Config, log *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(requestLogger(log))
	r.Use(cors)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Lets the frontend show the live bonus program limits instead of duplicating
	// them in the client bundle.
	r.Get("/api/config", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"cashbackPercent":      cfg.BonusCashbackPercent,
			"bonusSpendLimitPct":   cfg.BonusSpendLimitPct,
			"kopPerRuble":          money.KopPerRuble,
			"currency":             "RUB",
		})
	})

	r.Route("/api", func(r chi.Router) {
		productSvc := products.NewService(products.NewRepository(pool))
		customerSvc := customers.NewService(customers.NewRepository(pool))
		salesSvc := sales.NewService(sales.NewRepository(pool), pool,
			cfg.BonusCashbackPercent, cfg.BonusSpendLimitPct)

		products.NewHandler(productSvc, log).Mount(r)
		customers.NewHandler(customerSvc, log).Mount(r)
		sales.NewHandler(salesSvc, log).Mount(r)
	})

	return r
}

func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			log.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

// cors keeps the Vite dev server usable; in compose the frontend is served by nginx
// on the same origin, where CORS is not involved.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}