-- Активы, которые пользователь добавил сам: монеты (цена на Binance) и акции (цена на Yahoo Finance).
-- Список общий для всех, код актива один на всех: добавленный раз виден везде.
CREATE TABLE assets (
    code       TEXT        PRIMARY KEY,
    stock      BOOLEAN     NOT NULL,
    units      INT         NOT NULL CHECK (units BETWEEN 0 AND 9),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
