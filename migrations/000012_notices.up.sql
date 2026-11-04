-- Ежедневное напоминание внести траты: час по времени пользователя, NULL значит выключено.
ALTER TABLE users ADD COLUMN reminder_hour INT CHECK (reminder_hour BETWEEN 0 AND 23);

-- В какие дни напоминание уже отправлено.
CREATE TABLE daily_reminders (
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    day     DATE   NOT NULL,
    PRIMARY KEY (user_id, day)
);

-- Предупреждения о лимите категории: какой порог (80 или 100 процентов заложенного) в каком месяце уже отправлен.
CREATE TABLE limit_alerts (
    category_id BIGINT NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    month       DATE   NOT NULL,
    threshold   INT    NOT NULL CHECK (threshold IN (80, 100)),
    PRIMARY KEY (category_id, month, threshold)
);
