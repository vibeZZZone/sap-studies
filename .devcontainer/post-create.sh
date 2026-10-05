#!/usr/bin/env bash
# Prepares a codespace for shop-crm: starts Postgres, installs frontend dependencies,
# and creates the .env the API expects. Safe to re-run.
set -euo pipefail

cd "$(dirname "$0")/.."
PROJECT_DIR="shop-crm"

if [ ! -d "$PROJECT_DIR" ]; then
  echo "shop-crm/ not found in this checkout."
  echo "The project lives on the kilo/jovial-key-zkt branch; merge it into the branch"
  echo "the codespace was created from, or start the codespace on that branch."
  exit 1
fi

cd "$PROJECT_DIR"

# Postgres ships in the devcontainer image but does not start on its own.
if ! pg_isready -q 2>/dev/null; then
  sudo service postgresql start || true
  for _ in $(seq 1 30); do
    pg_isready -q 2>/dev/null && break
    sleep 1
  done
fi

# Dev credentials, matching .env.example. Only for a codespace, never for a real host.
sudo -u postgres psql -tc "SELECT 1 FROM pg_roles WHERE rolname='shop'" | grep -q 1 ||
  sudo -u postgres psql -c "CREATE ROLE shop LOGIN PASSWORD 'shop' SUPERUSER" || true
sudo -u postgres psql -tc "SELECT 1 FROM pg_database WHERE datname='shop_crm'" | grep -q 1 ||
  sudo -u postgres createdb -O shop shop_crm || true

[ -f .env ] || cp .env.example .env

echo "Installing frontend dependencies…"
cd frontend && npm install

cat <<'MESSAGE'

shop-crm is ready. Two terminals:

  terminal 1  cd shop-crm/backend  && go run ./cmd/api
  terminal 2  cd shop-crm/frontend && npm run dev

Then open http://localhost:5173 — the till. Migrations and demo data are applied
automatically when the API starts.

  make test-unit          unit tests, no Docker needed
  make lint               go vet + eslint
MESSAGE
