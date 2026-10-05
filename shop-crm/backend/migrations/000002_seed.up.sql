-- +migrate Up
-- Demo data mirroring the UI prototype. Seeded as a migration so that
-- `docker compose up` is enough to get a populated shop. A migration runs once,
-- so a container restart cannot duplicate these rows.
INSERT INTO products (name, price_kop, stock) VALUES
    ('Кофе молотый Arabica 250 г', 34900, 12),
    ('Чай зелёный Sencha 100 г',    27900, 8),
    ('Шоколад горький 70% 90 г',    19900, 15),
    ('Мёд цветочный 500 мл',        54900, 6),
    ('Сыр Гауда 200 г',             45900, 5),
    ('Масло оливковое 500 мл',      69900, 4),
    ('Печенье овсяное 300 г',       22900, 20),
    ('Кофе зёрна Эфиопия 1 кг',     129900, 3);

-- The seeded cards are 0001..0003, so continue the sequence after them.
SELECT setval('card_number_seq', 3, true);

INSERT INTO customers (card_number, name, phone, bonus_balance) VALUES
    ('5000-0001', 'Анна Ковалёва', '+7 900 111-22-33', 800),
    ('5000-0002', 'Игорь Смирнов',  '+7 900 222-33-44', 0),
    ('5000-0003', 'Мария Липатова', '+7 900 333-44-55', 1250);