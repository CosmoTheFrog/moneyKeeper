-- Кошельки и копилки живут вне бюджета: их остатки не входят в «К распределению».
ALTER TABLE accounts DROP CONSTRAINT accounts_kind_check;
ALTER TABLE accounts ADD CONSTRAINT accounts_kind_check CHECK (kind IN ('cash', 'debit', 'credit', 'wallet', 'piggy'));
ALTER TABLE accounts ADD COLUMN off_budget BOOLEAN GENERATED ALWAYS AS (kind IN ('wallet', 'piggy')) STORED;

-- Обмен (две записи в разных валютах) и комиссия. У перевода и обмена через границу бюджета есть категория.
DO $$
DECLARE c RECORD;
BEGIN
    FOR c IN
        SELECT conname FROM pg_constraint
        WHERE conrelid = 'transactions'::regclass AND contype = 'c' AND conname <> 'transactions_amount_minor_check'
    LOOP
        EXECUTE format('ALTER TABLE transactions DROP CONSTRAINT %I', c.conname);
    END LOOP;
END $$;

ALTER TABLE transactions ADD CONSTRAINT transactions_kind_check
    CHECK (kind IN ('expense', 'income', 'transfer', 'opening', 'exchange', 'fee'));

ALTER TABLE transactions ADD CONSTRAINT transactions_category_check CHECK (
    CASE kind
        WHEN 'expense' THEN category_id IS NOT NULL
        WHEN 'transfer' THEN TRUE
        WHEN 'exchange' THEN TRUE
        ELSE category_id IS NULL
    END
);

ALTER TABLE transactions ADD CONSTRAINT transactions_group_check CHECK (
    (kind NOT IN ('transfer', 'exchange') OR transfer_id IS NOT NULL)
    AND (transfer_id IS NULL OR kind IN ('transfer', 'exchange', 'fee', 'expense'))
);

ALTER TABLE transactions ADD CONSTRAINT transactions_sign_check CHECK (
    (kind IN ('expense', 'fee') AND amount_minor < 0)
    OR (kind IN ('income', 'opening') AND amount_minor > 0)
    OR kind IN ('transfer', 'exchange')
);
