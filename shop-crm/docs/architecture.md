# Архитектура / Architecture

Диаграммы к проекту shop-crm. Каждая подписана: что показывает и где в коде ей
соответствует.

> English diagrams and notes live in the same file; section headings are bilingual.

## 1. Контекст системы

Магазин работает в браузере кассира. Авторизации в MVP нет: один пользователь, один
компьютер, одна точка входа.

```mermaid
flowchart LR
    cashier["Кассир<br/>(браузер)"]
    api["shop-crm API<br/>Go + chi"]
    db[("PostgreSQL")]

    cashier -->|"JSON over HTTP"| api
    api -->|"pgx, SQL"| db

    subgraph runtime["docker compose"]
        api
        db
    end
```

Источник: `docker-compose.yml`, `shop-crm/backend/cmd/api/main.go`.

## 2. Контейнеры и их зоны ответственности

```mermaid
flowchart TB
    subgraph compose["docker compose"]
        pg["postgres:16-alpine<br/>данные и миграции"]
        be["backend<br/>Go 1.23, статический бинарник"]
        fe["frontend<br/>nginx + собранный Vite"]
    end

    be -->|"5432 · DATABASE_URL"| pg
    fe -->|"/api → proxy_pass"| be
    browser["Браузер<br/>localhost:5173"] --> fe
```

Ключевое: nginx проксирует `/api` на backend, поэтому в проде всё одноorigin и CORS
не участвует. В разработке Vite проксирует `/api` на `localhost:8080`.

Источник: `docker-compose.yml`, `frontend/nginx.conf`, `frontend/vite.config.ts`.

## 3. Слои бэкенда

Модули не знают друг о друге: общий доступ только через SQL и маршруты в `app`.

```mermaid
flowchart TB
    subgraph app["internal/app"]
        router["chi router<br/>app.go"]
    end

    subgraph modules["internal/modules"]
        p["products<br/>handler · service · repository"]
        c["customers<br/>handler · service · repository"]
        s["sales<br/>handler · service · repository"]
    end

    subgraph lib["internal/lib"]
        bonus["bonus.Calculate<br/>чистая функция"]
        money["money<br/>рубли ↔ копейки"]
        err["errors.AppError<br/>единый формат"]
    end

    dbpkg["internal/db<br/>пул pgx"]
    mig["migrations<br/>встроены в бинарник"]

    router --> p
    router --> c
    router --> s
    s --> bonus
    p --> money
    c --> err
    p --> err
    s --> err
    p --> dbpkg
    c --> dbpkg
    s --> dbpkg
    mig --> dbpkg
```

`handler` разбирает HTTP и пишет ошибки, `service` валидирует и применяет правила,
`repository` выполняет SQL. Правило одно: только `service` знает про бизнес-правила,
только `repository` — про SQL.

## 4. Схема данных

Деньги — целые числа в копейках. Инварианты продублированы в `CHECK`, чтобы они
держались даже при ошибке в коде.

```mermaid
erDiagram
    products ||--o{ sale_items : "sold as"
    customers ||--o{ sales : "buys"
    sales ||--|{ sale_items : "contains"

    products {
        serial id PK
        text name
        int price_kop "CHECK > 0"
        int stock "CHECK >= 0"
        bool archived "мягкое удаление"
        timestamptz created_at
    }

    customers {
        serial id PK
        text card_number UK "5000-0001, из card_number_seq"
        text name
        text phone UK
        int bonus_balance "CHECK >= 0, баллы"
        bigint total_spent_kop "CHECK >= 0"
        timestamptz created_at
    }

    sales {
        serial id PK "номер чека"
        int customer_id FK "null = без карты"
        int subtotal_kop
        int bonus_spent "баллы"
        int total_kop "к оплате деньгами"
        int bonus_earned "баллы"
        timestamptz created_at
    }

    sale_items {
        serial id PK
        int sale_id FK
        int product_id FK
        text name_snapshot "копия имени"
        int price_kop "копия цены"
        int qty "CHECK > 0"
    }
```

Две вещи, ради которых схема такая:

- `sale_items.name_snapshot` и `sale_items.price_kop` — снимок на момент продажи. Смена
  цены товара не переписывает историю чеков.
- `products.archived` — мягкое удаление. Строка остаётся, и старые чеки продолжают
  ссылаться на реальный товар.

Номер карты выдаёт последовательность `card_number_seq`, а не разбор максимума: первый
вариант ломался, как только в таблице появлялась карта с другим префиксом.

Источник: `backend/migrations/000001_init.up.sql`, `000002_seed.up.sql`.

## 5. Транзакция `POST /api/sales`

Самая важная последовательность в проекте. Блокировки идут в порядке `id`, иначе две
параллельные продажи, пересекающиеся по двум товарам, упираются в deadlock.

```mermaid
sequenceDiagram
    autonumber
    participant F as Фронтенд
    participant H as sales.Handler
    participant S as sales.Service
    participant B as bonus.Calculate
    participant DB as PostgreSQL

    F->>H: POST /api/sales<br/>customerId, bonusPercent, items
    H->>S: Create(ctx, input)
    S->>S: validateInput<br/>пустой чек, qty > 0, процент ≤ лимита

    S->>DB: BEGIN
    S->>DB: SELECT … FOR UPDATE<br/>products WHERE id = ANY(ids) ORDER BY id
    DB-->>S: заблокированные цены и остатки
    S->>S: проверка остатка для каждой строки

    opt есть customerId
        S->>DB: SELECT … FOR UPDATE customers
        DB-->>S: баланс бонусов
    end

    S->>B: Calculate(items, balance, percent)
    B-->>S: subtotal, bonusSpent, total, bonusEarned
    Note over B: чистая функция, БД не касается

    S->>DB: INSERT INTO sales
    loop каждая позиция
        S->>DB: UPDATE products SET stock = stock - qty<br/>WHERE stock >= qty
        S->>DB: INSERT INTO sale_items (снимок имени и цены)
    end
    opt есть customerId
        S->>DB: UPDATE customers<br/>bonus_balance, total_spent_kop
    end

    alt всё прошло
        S->>DB: COMMIT
        S-->>H: 201 + чек
        H-->>F: JSON чек
    else любая ошибка
        S->>DB: ROLLBACK
        S-->>H: 409 / 404 / 400
        H-->>F: error envelope
        Note over F: чек в интерфейсе не очищается
    end
```

Проверка остатка сделана дважды: в Go по заблокированным значениям и в `WHERE`
обновления. Если между чтением и записью состояние строки изменится, обновление
станет no-op, а транзакция откатится.

Источник: `backend/internal/modules/sales/service.go`, `repository.go`.

## 6. Расчёт чека

Единственное место, где живут правила бонусной программы. Функция чистая: без БД, без
часов, без случайных чисел.

```mermaid
flowchart TB
    start(["Calculate(params)"]) --> empty{"items пуст?"}
    empty -->|да| errEmpty["ErrEmptyCart"]
    empty -->|нет| pct{"percent в [0, лимит]?"}
    pct -->|нет| errPct["ошибка"]
    pct -->|да| sum["subtotal = Σ price_kop × qty"]
    sum --> card{"customerId задан?"}
    card -->|нет| noBonus["spent = 0<br/>earned = 0<br/>total = subtotal"]
    card -->|да| spend["spend_kop = subtotal × percent / 100<br/>min со стоимостью баланса"]
    spend --> round["spend_kop округляется<br/>до целого балла"]
    round --> total["total = subtotal − spend_kop"]
    total --> earn["earned = total × cashback / 100 / 100"]
    noBonus --> res(["Result"])
    earn --> res
```

Два округления, оба вниз, и оба важны:

- `spend_kop` округляется до целого балла, чтобы `total` всегда равнялся
  `subtotal − bonusSpent × 100`. Частичный балл остаётся у покупателя, деньги не
  исчезают.
- `earned` округляется вниз: магазин не начисляет обещанного лишнего.

Кэшбэк считается от суммы, **реально оплаченной деньгами**, а не от суммы чека. Если
чек на 1000 ₽ закрыт наполовину бонусами, начисляется 25 баллов, а не 50.

Источник: `backend/internal/lib/bonus/bonus.go`, тесты в `bonus_test.go`.

## 7. Фронтенд

```mermaid
flowchart TB
    app["App.tsx<br/>роутер + переключатель языка"]
    pages["pages<br/>Till · Products · Customers · Sales"]
    hooks["api/hooks.ts<br/>хуки TanStack Query"]
    client["api/client.ts<br/>fetch + ApiError"]
    types["api/types.ts<br/>ответы сервера"]
    errs["api/useApiError.ts<br/>code → перевод"]
    i18n["i18n<br/>ru.json · en.json"]
    money["lib/money.ts<br/>формат и предпросмотр"]
    comps["components<br/>Receipt · ProductTile · Modal"]

    app --> pages
    pages --> hooks
    pages --> comps
    pages --> money
    hooks --> client
    client --> types
    pages --> errs
    errs --> i18n
    pages --> i18n
```

`lib/money.ts` дублирует арифметику бэкенда, чтобы чек на кассе не «прыгал», пока
пользователь собирает его. Это предпросмотр, а не источник истины: после `POST /sales`
интерфейс показывает числа из ответа сервера.

## 8. Локализация ошибок

Ошибка доезжает до экрана двумя дорогами: стабильный код и английский текст.

```mermaid
sequenceDiagram
    participant S as sales.Service
    participant H as errors.Write
    participant C as api/client
    participant T as useApiError
    participant U as Экран

    S-->>H: AppError{code: INSUFFICIENT_STOCK,<br/>message: "Not enough stock…",<br/>params: {productId: "7"}}
    H-->>C: 409 { error: { code, message, params } }
    C->>C: бросить ApiError
    C-->>T: ApiError
    T->>T: t("errors.INSUFFICIENT_STOCK")
    alt есть перевод кода
        T-->>U: "Недостаточно товара «Кофе…»"
    else кода нет в бандле
        T-->>U: message сервера (английский)
    end
```

`params` несут контекст, которого нет в тексте: `productId` позволяет подставить
название товара из уже загруженного каталога. Названия всех кодов перечислены в
`errors.*` обоих бандлов.

Источник: `frontend/src/api/useApiError.ts`, `backend/internal/lib/errors/errors.go`,
ADR `docs/decisions/0003-localisation-strategy.md`.

## 9. Поток запроса целиком

Пример: кассир пробивает чек с картой и ползунком бонусов на 50 %.

```mermaid
sequenceDiagram
    autonumber
    actor U as Кассир
    participant T as TillPage
    participant P as previewSale
    participant Q as TanStack Query
    participant A as API
    participant X as транзакция продажи

    U->>T: тап по товару
    T->>T: добавить строку в чек
    U->>T: выбор карты
    T->>Q: useCustomers(search)
    Q->>A: GET /api/customers?search=…
    A-->>Q: [{ id, cardNumber, bonusBalance }]
    U->>T: сдвиг ползунка на 50 %
    T->>P: subtotal, balance, percent
    P-->>T: предпросмотр (тот же расчёт, что на сервере)
    U->>T: «Пробить чек»
    T->>Q: useCreateSale
    Q->>A: POST /api/sales
    A->>X: BEGIN, FOR UPDATE, расчёт, запись
    X-->>A: COMMIT
    A-->>Q: 201 { sale }
    Q->>Q: инвалидировать sales, products, customers
    Q-->>T: числа сервера
    T->>T: очистить чек, показать итог
```

При 409 чек остаётся на экране: кассир правит его, а не собирает заново.

## 10. Потоки тестов

```mermaid
flowchart LR
    subgraph unit["go test ./internal/... — без Docker"]
        b["bonus_test.go<br/>округления, лимиты, пустой чек"]
        m["money_test.go<br/>конвертация и формат"]
    end

    subgraph integ["go test ./tests/... — Postgres"]
        tc["testcontainers<br/>или TEST_DATABASE_URL"]
        sc["sales_test.go<br/>критерии приёмки"]
        rc["sales_race_test.go<br/>гонка за последнюю единицу"]
        pt["products_test.go<br/>жизненный цикл, валидация"]
        ct["customers_test.go<br/>карта, поиск, 409"]
    end

    unit -.->|"одна и та же арифметика"| integ
```

Расчёт чека покрыт юнит-тестами без БД, а интеграционные проверяют то, что юнит-тесты
не видят: транзакцию, блокировки и HTTP-контракт.
