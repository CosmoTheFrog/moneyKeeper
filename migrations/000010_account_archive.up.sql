-- Закрытый счёт скрывается из списков, а записи по нему остаются в истории.
ALTER TABLE accounts ADD COLUMN archived_at TIMESTAMPTZ;
