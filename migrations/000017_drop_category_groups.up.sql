-- Группы категорий убраны: остались только категории. Перенос остатка уже хранится в categories.rollover.
DROP INDEX IF EXISTS categories_group_idx;
ALTER TABLE categories DROP COLUMN group_id;
DROP TABLE category_groups;
