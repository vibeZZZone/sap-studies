-- +migrate Up
CREATE TABLE products (
    id         serial PRIMARY KEY,
    name       text NOT NULL,
    price_kop  integer NOT NULL CHECK (price_kop > 0),
    stock      integer NOT NULL CHECK (stock >= 0),
    archived   boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE customers (
    id              serial PRIMARY KEY,
    card_number     text NOT NULL UNIQUE,
    name            text NOT NULL,
    phone           text NOT NULL UNIQUE,
    bonus_balance   integer NOT NULL CHECK (bonus_balance >= 0),
    total_spent_kop bigint NOT NULL DEFAULT 0 CHECK (total_spent_kop >= 0),
    created_at      timestamptz NOT NULL DEFAULT now()
);

-- Card numbers are 5000-0001, 5000-0002, … The sequence is the single source of
-- truth for the number: parsing the highest existing card would break as soon as a
-- card is archived or inserted by hand.
CREATE SEQUENCE card_number_seq START 1;

CREATE TABLE sales (
    id           serial PRIMARY KEY,
    customer_id  integer REFERENCES customers(id),
    subtotal_kop integer NOT NULL CHECK (subtotal_kop >= 0),
    bonus_spent  integer NOT NULL CHECK (bonus_spent >= 0),
    total_kop    integer NOT NULL CHECK (total_kop >= 0),
    bonus_earned integer NOT NULL CHECK (bonus_earned >= 0),
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sale_items (
    id            serial PRIMARY KEY,
    sale_id       integer NOT NULL REFERENCES sales(id) ON DELETE CASCADE,
    product_id    integer NOT NULL REFERENCES products(id),
    name_snapshot text NOT NULL,
    price_kop     integer NOT NULL CHECK (price_kop > 0),
    qty           integer NOT NULL CHECK (qty > 0)
);

CREATE INDEX idx_customers_phone       ON customers(phone);
CREATE INDEX idx_customers_card_number ON customers(card_number);
CREATE INDEX idx_sales_created_at_desc ON sales(created_at DESC);
CREATE INDEX idx_sale_items_sale_id    ON sale_items(sale_id);
CREATE INDEX idx_products_archived     ON products(archived);