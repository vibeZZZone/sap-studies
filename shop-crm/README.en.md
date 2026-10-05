# Mini shop CRM

A fullstack study project: till, stock control, loyalty cards for customers, and sales
history. The backend is the single source of truth for money and stock.

> Русская версия: [README.md](README.md) · Architecture decisions:
> [docs/decisions](docs/decisions)

## Features

- **Till** — product tiles with price and stock, a receipt with adjustable quantities,
  customer lookup by card, a slider to spend bonuses, and a "complete sale" action.
- **Customers** — server-side search by name, phone, and card number; cards issued
  sequentially in the `5000-0001` format.
- **Products** — add, receive stock (+10), delete with soft archiving.
- **Sales** — receipt history, 20 per page, with line items and bonus amounts.
- **Localisation** — Russian and English with a header switcher; the choice persists.

## Running it

```bash
cp .env.example .env
docker compose up --build
```

- Till: <http://localhost:5173>
- API: <http://localhost:8080>, health check: <http://localhost:8080/health>

Migrations and demo data (8 products, 3 customers) are applied automatically when the
backend starts. Restarting does not duplicate them: the demo data is a migration, and a
migration runs once.

### Without Docker

```bash
cd backend && go run ./cmd/api           # needs a Postgres reachable via DATABASE_URL
cd frontend && npm install && npm run dev  # http://localhost:5173, proxies /api
```

## Environment variables

| Variable | Default | Purpose |
|---|---|---|
| `POSTGRES_DB` | `shop_crm` | database name |
| `POSTGRES_USER` | `shop` | database user |
| `POSTGRES_PASSWORD` | `shop` | database password |
| `POSTGRES_PORT` | `5432` | exposed Postgres port |
| `BACKEND_PORT` | `8080` | exposed API port |
| `FRONTEND_PORT` | `5173` | exposed UI port |
| `LOG_LEVEL` | `info` | API log level |
| `BONUS_CASHBACK_PERCENT` | `5` | cashback on the amount actually paid in cash, floored to whole points |
| `BONUS_SPEND_LIMIT_PERCENT` | `50` | maximum share of a receipt that bonuses may cover |
| `SEED_ON_STARTUP` | `true` | apply demo data on start |

Secrets stay out of git: `.env` is ignored and only `.env.example` is committed.

## Business rules

The server calculates; the client only previews.

1. Money is stored as whole kopecks. `float` is never used for money.
2. Cashback is 5 % of what the customer actually pays in cash, after bonuses, floored to
   a whole point. One point is worth 1 ₽.
3. Bonuses may cover at most 50 % of the subtotal, rounded down.
4. No more than the card holds can be spent.
5. Bonuses are only earned and spent with a card; a sale without a card has none.
6. Stock cannot go below zero, and stock never goes negative.
7. A receipt is immutable once created; no refunds in the MVP.
8. Price and product name are copied onto the receipt lines, so a later price change does
   not rewrite history.

## API

Base path `/api`, JSON. Errors always look like
`{ "error": { "code": "STRING", "message": "text", "params": {} } }`.

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/config` | bonus program parameters |
| GET | `/api/products` | products, archived ones excluded |
| POST | `/api/products` | create a product |
| POST | `/api/products/:id/restock` | receive stock, body `{ "qty": 10 }` |
| DELETE | `/api/products/:id` | archive a product |
| GET | `/api/customers?search=` | customers with server-side search |
| POST | `/api/customers` | issue a card |
| GET | `/api/sales?page=1` | receipt history with line items |
| POST | `/api/sales` | complete a sale |

### Example: completing a sale

```bash
curl -X POST http://localhost:8080/api/sales \
  -H 'Content-Type: application/json' \
  -d '{
        "customerId": 1,
        "bonusPercent": 50,
        "items": [{ "productId": 1, "qty": 2 }]
      }'
```

```json
{
  "id": 1,
  "customerId": 1,
  "subtotalKop": 69800,
  "subtotal": "698 ₽",
  "bonusSpent": 349,
  "totalKop": 34900,
  "total": "349 ₽",
  "bonusEarned": 17
}
```

The client sends no amounts: the server computes them from the rules above and returns
what actually happened.

### Example: insufficient stock

```json
{
  "error": {
    "code": "INSUFFICIENT_STOCK",
    "message": "Not enough stock for this product",
    "params": { "productId": "7" }
  }
}
```

The client translates `INSUFFICIENT_STOCK` into the UI language and substitutes the
product name from the catalogue it already has. Stock and bonuses are untouched: the
transaction rolls back as a whole.

### Error codes

`VALIDATION_ERROR`, `EMPTY_CART`, `PRODUCT_NOT_FOUND`, `CUSTOMER_NOT_FOUND`,
`PHONE_ALREADY_EXISTS`, `INSUFFICIENT_STOCK`, `NOT_FOUND`, `INTERNAL_ERROR`.

## Project layout

```
shop-crm/
  docker-compose.yml
  .env.example
  Makefile
  backend/
    cmd/api/main.go            entrypoint, migrations, graceful shutdown
    migrations/                versioned .sql, embedded in the binary
    internal/
      app/app.go               HTTP router wiring
      config/config.go         env and bonus program constants
      db/db.go                 pgx pool that waits for Postgres readiness
      lib/money/               rubles to kopecks, formatting
      lib/bonus/               calculateSale — pure receipt calculation
      lib/errors/              AppError and the unified error response
      modules/products/        handler, service, repository
      modules/customers/
      modules/sales/           sale transaction with FOR UPDATE
    tests/                     integration tests on testcontainers
  frontend/
    src/api/                   client, TanStack Query hooks, error translation
    src/i18n/                  ru/en bundles, language switcher
    src/pages/                 Till, Customers, Products, Sales
    src/components/            Receipt, ProductTile, Modal, AsyncBoundary
    src/lib/money.ts           formatting and receipt preview
  docs/decisions/              ADRs: why Go, the bonus percent, i18n, locking
```

## Tests

```bash
make test-unit          # receipt calculation and money unit tests, no Docker
make test-integration   # integration tests, needs Docker
make test               # everything
make lint               # go vet + eslint
make typecheck          # go build + tsc
```

Where Docker is unavailable, run the integration tests against your own Postgres:

```bash
TEST_DATABASE_URL="postgres://shop:shop@localhost:5432/shop_crm_test?sslmode=disable" \
  go test ./tests/... -count=1
```

The tests apply the migrations themselves, but the database must be empty:
`DROP SCHEMA public CASCADE; CREATE SCHEMA public;` on that database is enough.

The integration tests spin up a real Postgres through testcontainers, apply the
migrations, and check among other things that:

- the spec example computes correctly: 1000 ₽, 800 points on the card,
  `bonusPercent: 50` → 500 spent, 500 ₽ paid, 25 earned;
- an over-stock request returns 409 with stock and bonuses unchanged;
- two parallel requests for the last unit yield one 201 and one 409;
- a price change does not alter an earlier receipt;
- pagination returns 20 receipts per page, newest first.

## Out of scope

Authentication and roles, refunds, discounts and promotions, barcodes and scanners,
reports and charts, multi-store support, receipt printing, and cash-register hardware
integrations.

The architecture allows them: authentication slots in as middleware in `internal/app`,
and reports read the same tables, so no breaking schema changes are needed.