-- Счёт с несколькими валютами (как Revolut или Binance): записи в любой поддерживаемой валюте.
-- У обычного счёта по-прежнему одна валюта, currency остаётся основной валютой счёта.
ALTER TABLE accounts ADD COLUMN multi_currency BOOLEAN NOT NULL DEFAULT FALSE;
