-- Расписание платежей: куда и когда нужно заплатить. Транзакции автоматически не создаются.
CREATE TABLE schedules (
    id                BIGSERIAL PRIMARY KEY,
    budget_id         BIGINT      NOT NULL REFERENCES budgets (id) ON DELETE CASCADE,
    name              TEXT        NOT NULL,
    category_id       BIGINT      NOT NULL REFERENCES categories (id),
    account_id        BIGINT REFERENCES accounts (id),
    currency          TEXT        NOT NULL,
    -- exact: точная сумма, approx: примерная (допуск 7,5%), range: диапазон от amount_minor до amount_max_minor.
    amount_kind       TEXT        NOT NULL CHECK (amount_kind IN ('exact', 'approx', 'range')),
    amount_minor      BIGINT      NOT NULL CHECK (amount_minor > 0),
    amount_max_minor  BIGINT,
    -- Раз в every_months месяцев, начиная с start_month, в числа days (0 значит последний день месяца).
    every_months      INT         NOT NULL CHECK (every_months BETWEEN 1 AND 24),
    days              SMALLINT[]  NOT NULL CHECK (cardinality(days) BETWEEN 1 AND 5),
    start_month       DATE        NOT NULL CHECK (EXTRACT(DAY FROM start_month) = 1),
    -- Платёж на выходной: none оставить, back перенести на предыдущий будний день, forward на следующий.
    weekend_shift     TEXT        NOT NULL CHECK (weekend_shift IN ('none', 'back', 'forward')),
    remind_before     INT         NOT NULL CHECK (remind_before BETWEEN 0 AND 7),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((amount_kind = 'range') = (amount_max_minor IS NOT NULL)),
    CHECK (amount_max_minor IS NULL OR amount_max_minor >= amount_minor)
);

CREATE INDEX schedules_budget_idx ON schedules (budget_id);

-- Пропущенный пользователем платёж этой даты.
CREATE TABLE schedule_skips (
    schedule_id BIGINT NOT NULL REFERENCES schedules (id) ON DELETE CASCADE,
    due_on      DATE   NOT NULL,
    PRIMARY KEY (schedule_id, due_on)
);

-- Какие напоминания уже отправлены: lead это за сколько дней до платежа (0 в день платежа).
CREATE TABLE schedule_reminders (
    schedule_id BIGINT      NOT NULL REFERENCES schedules (id) ON DELETE CASCADE,
    due_on      DATE        NOT NULL,
    lead        INT         NOT NULL,
    sent_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (schedule_id, due_on, lead)
);