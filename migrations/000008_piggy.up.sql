-- Копилка: цель (сумма и дата, необязательные) и конверт в группе «Цели накопления», из которого она пополняется.
ALTER TABLE accounts ADD COLUMN target_minor BIGINT CHECK (target_minor > 0);
ALTER TABLE accounts ADD COLUMN target_date DATE;
ALTER TABLE accounts ADD COLUMN category_id BIGINT REFERENCES categories (id);
