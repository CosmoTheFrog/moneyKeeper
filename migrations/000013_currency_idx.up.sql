-- Портфель ищет записи бюджета по валюте монеты (CoinActivity), раньше для этого был обход всех записей бюджета.
CREATE INDEX transactions_budget_currency_idx ON transactions (budget_id, currency) WHERE deleted_at IS NULL;
