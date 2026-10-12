-- Группы категорий становятся сущностью бюджета: можно заводить свои. У стартовых групп есть ключ,
-- у созданных пользователем ключа нет.
CREATE TABLE category_groups (
    id         BIGSERIAL PRIMARY KEY,
    budget_id  BIGINT      NOT NULL REFERENCES budgets (id) ON DELETE CASCADE,
    key        TEXT,
    name       TEXT        NOT NULL,
    position   INT         NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (budget_id, name)
);

CREATE UNIQUE INDEX category_groups_key_idx ON category_groups (budget_id, key) WHERE key IS NOT NULL;

INSERT INTO category_groups (budget_id, key, name, position)
SELECT b.id, g.key, g.name, g.position
FROM budgets b
CROSS JOIN (VALUES
    ('critical', 'Критичные', 1),
    ('debt', 'Долги', 2),
    ('daily', 'Ежедневные', 3),
    ('rainy', 'На чёрный день', 4),
    ('goal', 'Цели накопления', 5)
) AS g (key, name, position);

ALTER TABLE categories ADD COLUMN group_id BIGINT REFERENCES category_groups (id);

UPDATE categories c SET group_id = g.id
FROM category_groups g
WHERE g.budget_id = c.budget_id AND g.key = c.group_key;

ALTER TABLE categories ALTER COLUMN group_id SET NOT NULL;
ALTER TABLE categories DROP COLUMN group_key;

CREATE INDEX categories_group_idx ON categories (group_id);
