CREATE TABLE categories (
    id          BIGSERIAL PRIMARY KEY,
    budget_id   BIGINT      NOT NULL REFERENCES budgets (id) ON DELETE CASCADE,
    group_key   TEXT        NOT NULL CHECK (group_key IN ('critical', 'debt', 'daily', 'rainy', 'goal')),
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    hidden      BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (budget_id, name)
);

CREATE SEQUENCE transfer_ids;

CREATE TABLE transactions (
    id           BIGSERIAL PRIMARY KEY,
    budget_id    BIGINT      NOT NULL REFERENCES budgets (id) ON DELETE CASCADE,
    account_id   BIGINT      NOT NULL REFERENCES accounts (id),
    category_id  BIGINT REFERENCES categories (id),
    kind         TEXT        NOT NULL CHECK (kind IN ('expense', 'income', 'transfer', 'opening')),
    amount_minor BIGINT      NOT NULL CHECK (amount_minor <> 0),
    currency     TEXT        NOT NULL,
    occurred_on  DATE        NOT NULL,
    note         TEXT        NOT NULL DEFAULT '',
    transfer_id  BIGINT,
    created_by   BIGINT      NOT NULL REFERENCES users (id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ,
    CHECK ((kind = 'expense') = (category_id IS NOT NULL)),
    CHECK ((kind = 'transfer') = (transfer_id IS NOT NULL)),
    CHECK (
        (kind = 'expense' AND amount_minor < 0)
        OR (kind IN ('income', 'opening') AND amount_minor > 0)
        OR kind = 'transfer'
    )
);

CREATE INDEX transactions_budget_day_idx ON transactions (budget_id, occurred_on DESC, id DESC) WHERE deleted_at IS NULL;
CREATE INDEX transactions_account_idx ON transactions (account_id) WHERE deleted_at IS NULL;
CREATE INDEX transactions_transfer_idx ON transactions (transfer_id) WHERE transfer_id IS NOT NULL;

CREATE TABLE transaction_events (
    id             BIGSERIAL PRIMARY KEY,
    transaction_id BIGINT      NOT NULL REFERENCES transactions (id) ON DELETE CASCADE,
    user_id        BIGINT      NOT NULL REFERENCES users (id),
    action         TEXT        NOT NULL CHECK (action IN ('create', 'update', 'delete')),
    before         JSONB,
    after          JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX transaction_events_tx_idx ON transaction_events (transaction_id, id);
