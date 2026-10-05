.PHONY: help up down logs build test test-unit test-integration lint typecheck migrate

help:
	@echo "make up               поднять всё приложение (docker compose)"
	@echo "make down             остановить контейнеры"
	@echo "make logs             логи сервисов"
	@echo "make test             все тесты бэкенда (нужен Docker для интеграционных)"
	@echo "make test-unit        только юнит-тесты, без Docker"
	@echo "make test-integration интеграционные тесты на testcontainers"
	@echo "make lint             go vet + eslint"
	@echo "make typecheck        tsc --noEmit + go build"

up:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f

build:
	docker compose build

migrate:
	cd backend && go run ./cmd/api

test:
	cd backend && go test ./...

test-unit:
	cd backend && go test ./internal/...

test-integration:
	cd backend && go test ./tests/... -count=1

lint:
	cd backend && go vet ./...
	cd frontend && npm run lint

typecheck:
	cd backend && go build ./...
	cd frontend && npm run typecheck