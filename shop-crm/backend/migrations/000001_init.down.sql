-- +migrate Down
DROP TABLE IF EXISTS sale_items;
DROP TABLE IF EXISTS sales;
DROP TABLE IF EXISTS customers;
DROP SEQUENCE IF EXISTS card_number_seq;
DROP TABLE IF EXISTS products;