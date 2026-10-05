# Мини-CRM продуктового магазина / Shop CRM

Учебное fullstack-приложение: касса, складской учёт, бонусные карты покупателей и
история продаж. Источник истины по всем деньгам и остаткам — бэкенд.

> English version: [README.en.md](README.en.md) · Architecture decisions:
> [docs/decisions](docs/decisions)

## Возможности

- **Касса** — плитки товаров с ценой и остатком, чек с изменением количества, выбор
  покупателя по карте, списание бонусов ползунком, «Пробить чек».
- **Покупатели** — поиск по имени, телефону и номеру карты на сервере, выдача карты
  последовательно в формате `5000-0001`.
- **Товары** — добавление, приёмка (+10), удаление с мягким архивированием.
- **Продажи** — история чеков по 20 на страницу, состав чека, списанные и начисленные
  бонусы.
- **Локализация** — русский и английский, переключатель в шапке, язык сохраняется.

## Запуск

```bash
cp .env.example .env
docker compose up --build
```

- Касса: <http://localhost:5173>
- API: <http://localhost:8080>, проверка состояния — <http://localhost:8080/health>

Миграции и демо-данные (8 товаров, 3 покупателя) применяются автоматически при старте
бэкенда. Повторный запуск не дублирует их: демо-данные оформлены как миграция, которая
применяется один раз.

### Без Docker

```bash
cd backend && go run ./cmd/api          # нужна доступная Postgres из DATABASE_URL
cd frontend && npm install && npm run dev # http://localhost:5173, проксирует /api
```

## Переменные окружения

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `POSTGRES_DB` | `shop_crm` | имя базы |
| `POSTGRES_USER` | `shop` | пользователь базы |
| `POSTGRES_PASSWORD` | `shop` | пароль базы |
| `POSTGRES_PORT` | `5432` | порт Postgres наружу |
| `BACKEND_PORT` | `8080` | порт API наружу |
| `FRONTEND_PORT` | `5173` | порт интерфейса наружу |
| `LOG_LEVEL` | `info` | уровень логов API |
| `BONUS_CASHBACK_PERCENT` | `5` | кэшбэк от оплаченного деньгами, вниз до целого балла |
| `BONUS_SPEND_LIMIT_PERCENT` | `50` | максимум чека, который можно закрыть бонусами |
| `SEED_ON_STARTUP` | `true` | применять демо-данные при старте |

Секреты не коммитятся: `.env` в `.gitignore`, в репозитории лежит только `.env.example`.

## Бизнес-правила

Считает сервер, клиент только показывает предпросмотр.

1. Деньги хранятся целыми числами в копейках. `float` не используется для денег.
2. Кэшбэк 5 % от суммы, реально оплаченной деньгами (после вычета бонусов), округление
   вниз до целого балла. 1 балл = 1 ₽.
3. Бонусами закрывается не больше 50 % суммы до вычета, округление вниз.
4. Нельзя списать больше, чем есть на карте.
5. Бонусы начисляются и списываются только с картой; чек без карты — без бонусов.
6. Нельзя продать больше складского остатка; остаток не уходит в минус.
7. Чек неизменяем после создания; возвратов в MVP нет.
8. Цена и название товара копируются в строки чека, поэтому смена цены не меняет историю.

## API

Базовый путь `/api`, формат JSON. Ошибки всегда вида
`{ "error": { "code": "STRING", "message": "текст", "params": {} } }`.

| Метод | Путь | Назначение |
|---|---|---|
| GET | `/api/config` | параметры бонусной программы |
| GET | `/api/products` | список товаров без архивных |
| POST | `/api/products` | создать товар |
| POST | `/api/products/:id/restock` | приёмка, тело `{ "qty": 10 }` |
| DELETE | `/api/products/:id` | архивировать товар |
| GET | `/api/customers?search=` | покупатели с серверным поиском |
| POST | `/api/customers` | выдать карту |
| GET | `/api/sales?page=1` | история чеков с позициями |
| POST | `/api/sales` | пробить чек |

### Пример: пробить чек

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

Клиент не передаёт суммы: сервер сам считает их по правилам выше и возвращает факт.

### Пример: ошибка нехватки остатка

```json
{
  "error": {
    "code": "INSUFFICIENT_STOCK",
    "message": "Not enough stock for this product",
    "params": { "productId": "7" }
  }
}
```

Код `INSUFFICIENT_STOCK` клиент переводит на язык интерфейса и подставляет название
товара из уже загруженного каталога. Остатки и бонусы при этом не меняются: транзакция
откатывается целиком.

### Коды ошибок

`VALIDATION_ERROR`, `EMPTY_CART`, `PRODUCT_NOT_FOUND`, `CUSTOMER_NOT_FOUND`,
`PHONE_ALREADY_EXISTS`, `INSUFFICIENT_STOCK`, `NOT_FOUND`, `INTERNAL_ERROR`.

## Структура проекта

```
shop-crm/
  docker-compose.yml
  .env.example
  Makefile
  backend/
    cmd/api/main.go            точка входа, миграции, graceful shutdown
    migrations/                версионированные .sql, встроены в бинарник
    internal/
      app/app.go               сборка HTTP-маршрутов
      config/config.go         env и константы бонусной программы
      db/db.go                 пул pgx с ожиданием готовности Postgres
      lib/money/               рубли ↔ копейки, форматирование
      lib/bonus/               calculateSale — чистая функция расчёта чека
      lib/errors/              AppError и единый формат ответа
      modules/products/        handler, service, repository
      modules/customers/
      modules/sales/           транзакция продажи с FOR UPDATE
    tests/                     интеграционные тесты на testcontainers
  frontend/
    src/api/                   клиент, хуки TanStack Query, перевод ошибок
    src/i18n/                  бандлы ru/en, переключатель языка
    src/pages/                 Till, Customers, Products, Sales
    src/components/            Receipt, ProductTile, Modal, AsyncBoundary
    src/lib/money.ts           форматирование и предпросмотр чека
  docs/decisions/              ADR: почему Go, процент бонусов, i18n, блокировки
```

## Тесты

```bash
make test-unit          # юнит-тесты расчёта чека и денег, без Docker
make test-integration   # интеграционные тесты, нужен Docker
make test               # всё
make lint               # go vet + eslint
make typecheck          # go build + tsc
```

Если Docker недоступен, интеграционные тесты можно гонять на своей Postgres:

```bash
TEST_DATABASE_URL="postgres://shop:shop@localhost:5432/shop_crm_test?sslmode=disable" \
  go test ./tests/... -count=1
```

Тесты применяют миграции сами, но база должна быть пустой: `DROP SCHEMA public CASCADE;
CREATE SCHEMA public;` в этой базе достаточно.

Интеграционные тесты поднимают настоящий Postgres через testcontainers, накатывают
миграции и проверяют, в том числе:

- расчёт по критерию ТЗ: 1000 ₽, на карте 800 баллов, `bonusPercent: 50` →
  списано 500, оплачено 500 ₽, начислено 25;
- 409 при нехватке остатка, причём остатки и бонусы не меняются;
- два параллельных запроса за последнюю единицу: один 201, второй 409;
- смена цены товара не меняет ранее созданные чеки;
- пагинация по 20 чеков и порядок от новых к старым.

## Вне рамок MVP

Авторизация и роли, возвраты, скидки и акции, штрихкоды и сканер, отчёты и графики,
мультимагазин, печать чеков, интеграции с кассовым оборудованием.

Архитектура их допускает: авторизация добавляется middleware в `internal/app`, отчёты
читают те же таблицы, а несовместимых изменений схемы не потребуется.