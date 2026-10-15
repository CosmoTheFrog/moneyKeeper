-- Бюджет по месяцам: сколько заложено на категорию в месяце. Остальное (потрачено, остаток, «К распределению»)
-- считается из транзакций и этой таблицы.
CREATE TABLE category_months (
    category_id    BIGINT NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    month          DATE   NOT NULL CHECK (EXTRACT(DAY FROM month) = 1),
    budgeted_minor BIGINT NOT NULL CHECK (budgeted_minor >= 0),
    PRIMARY KEY (category_id, month)
);

-- Перенос остатка категории в следующий месяц: у накопительных групп включён с самого начала.
ALTER TABLE categories ADD COLUMN rollover BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE categories c SET rollover = TRUE
FROM category_groups g
WHERE g.id = c.group_id AND g.key IN ('rainy', 'goal');
