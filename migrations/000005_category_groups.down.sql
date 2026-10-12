-- Созданные пользователем группы при откате сливаются в «Ежедневные».
ALTER TABLE categories ADD COLUMN group_key TEXT CHECK (group_key IN ('critical', 'debt', 'daily', 'rainy', 'goal'));

UPDATE categories c SET group_key = COALESCE(g.key, 'daily')
FROM category_groups g
WHERE g.id = c.group_id;

ALTER TABLE categories ALTER COLUMN group_key SET NOT NULL;
ALTER TABLE categories DROP COLUMN group_id;

DROP TABLE category_groups;
