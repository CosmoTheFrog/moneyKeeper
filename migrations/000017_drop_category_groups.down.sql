-- При откате все категории попадают в одну группу.
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

INSERT INTO category_groups (budget_id, name, position) SELECT id, 'Категории', 1 FROM budgets;

ALTER TABLE categories ADD COLUMN group_id BIGINT REFERENCES category_groups (id);
UPDATE categories c SET group_id = g.id FROM category_groups g WHERE g.budget_id = c.budget_id;
ALTER TABLE categories ALTER COLUMN group_id SET NOT NULL;
CREATE INDEX categories_group_idx ON categories (group_id);
