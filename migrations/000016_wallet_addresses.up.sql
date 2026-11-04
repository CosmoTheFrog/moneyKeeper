-- Адреса публичных криптокошельков, привязанные к счетам-кошелькам. Согласие на отправку адреса во внешний источник
-- записывается при привязке. Ключей и seed-фраз здесь нет.
CREATE TABLE wallet_addresses (
    id         BIGSERIAL PRIMARY KEY,
    account_id BIGINT      NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    chain      TEXT        NOT NULL CHECK (chain IN ('eth', 'tron', 'sol')),
    address    TEXT        NOT NULL,
    consent_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (account_id, chain, address)
);

-- Переводы из сети, уже внесённые в записи: по ним повторное чтение пропускает дубли.
CREATE TABLE chain_imported (
    account_id  BIGINT NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    external_id TEXT   NOT NULL,
    PRIMARY KEY (account_id, external_id)
);
