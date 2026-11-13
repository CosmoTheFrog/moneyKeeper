-- Откат возможен, пока нет пользователей без Telegram: иначе SET NOT NULL откажет.
DROP TABLE sessions;
ALTER TABLE users DROP CONSTRAINT users_identity_check;
ALTER TABLE users DROP COLUMN apple_sub;
ALTER TABLE users ALTER COLUMN telegram_id SET NOT NULL;
