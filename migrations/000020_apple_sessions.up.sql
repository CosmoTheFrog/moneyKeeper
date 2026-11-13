-- Вход через Apple для нативного приложения: пользователь может не иметь Telegram, но обязательно имеет хотя бы один
-- способ входа. Сессии хранят только хэш токена: сам токен знает лишь приложение.
ALTER TABLE users ALTER COLUMN telegram_id DROP NOT NULL;
ALTER TABLE users ADD COLUMN apple_sub TEXT UNIQUE;
ALTER TABLE users ADD CONSTRAINT users_identity_check CHECK (telegram_id IS NOT NULL OR apple_sub IS NOT NULL);

CREATE TABLE sessions (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash BYTEA       NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX sessions_user_idx ON sessions (user_id);
