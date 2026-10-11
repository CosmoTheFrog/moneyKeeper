CREATE TABLE users (
    id               BIGSERIAL PRIMARY KEY,
    telegram_id      BIGINT      NOT NULL UNIQUE,
    first_name       TEXT        NOT NULL,
    timezone         TEXT        NOT NULL,
    active_budget_id BIGINT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE budgets (
    id            BIGSERIAL PRIMARY KEY,
    name          TEXT        NOT NULL,
    base_currency TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE budget_members (
    budget_id  BIGINT      NOT NULL REFERENCES budgets (id) ON DELETE CASCADE,
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (budget_id, user_id)
);

ALTER TABLE users
    ADD CONSTRAINT users_active_budget_fk FOREIGN KEY (active_budget_id) REFERENCES budgets (id) ON DELETE SET NULL;

CREATE TABLE accounts (
    id         BIGSERIAL PRIMARY KEY,
    budget_id  BIGINT      NOT NULL REFERENCES budgets (id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    kind       TEXT        NOT NULL CHECK (kind IN ('cash', 'debit', 'credit')),
    currency   TEXT        NOT NULL,
    is_default BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX accounts_budget_idx ON accounts (budget_id);
CREATE UNIQUE INDEX accounts_one_default_per_budget ON accounts (budget_id) WHERE is_default;
