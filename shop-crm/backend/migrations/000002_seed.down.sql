-- +migrate Down
DELETE FROM customers WHERE card_number IN ('5000-0001', '5000-0002', '5000-0003');
DELETE FROM products WHERE name IN (
    'Кофе молотый Arabica 250 г', 'Чай зелёный Sencha 100 г', 'Шоколад горький 70% 90 г',
    'Мёд цветочный 500 мл', 'Сыр Гауда 200 г', 'Масло оливковое 500 мл',
    'Печенье овсяное 300 г', 'Кофе зёрна Эфиопия 1 кг'
);