-- Режим конвертов (заложенные суммы, «К распределению», лимиты): по умолчанию у бюджета только траты по категориям.
ALTER TABLE budgets ADD COLUMN envelopes_enabled BOOLEAN NOT NULL DEFAULT false;
