-- Запомненные соответствия при импорте выписок: продавец или код MCC и выбранная категория.
-- key: «d:» и название продавца в нижнем регистре либо «s:» и название из файла (например, MCC 5411).
CREATE TABLE import_rules (
    budget_id   BIGINT      NOT NULL REFERENCES budgets (id) ON DELETE CASCADE,
    key         TEXT        NOT NULL,
    category_id BIGINT      NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (budget_id, key)
);
