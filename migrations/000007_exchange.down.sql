-- Обмены, комиссии и переводы с категорией при откате удаляются: старая схема их не знает.
DELETE FROM transaction_events WHERE transaction_id IN (SELECT id FROM transactions WHERE kind IN ('exchange', 'fee'));
DELETE FROM transactions WHERE kind IN ('exchange', 'fee');
UPDATE transactions SET category_id = NULL WHERE kind = 'transfer';
DELETE FROM transactions WHERE kind = 'expense' AND transfer_id IS NOT NULL;

ALTER TABLE transactions DROP CONSTRAINT transactions_kind_check;
ALTER TABLE transactions DROP CONSTRAINT transactions_category_check;
ALTER TABLE transactions DROP CONSTRAINT transactions_group_check;
ALTER TABLE transactions DROP CONSTRAINT transactions_sign_check;

ALTER TABLE transactions ADD CONSTRAINT transactions_kind_check CHECK (kind IN ('expense', 'income', 'transfer', 'opening'));
ALTER TABLE transactions ADD CONSTRAINT transactions_check CHECK ((kind = 'expense') = (category_id IS NOT NULL));
ALTER TABLE transactions ADD CONSTRAINT transactions_check1 CHECK ((kind = 'transfer') = (transfer_id IS NOT NULL));
ALTER TABLE transactions ADD CONSTRAINT transactions_check2 CHECK (
    (kind = 'expense' AND amount_minor < 0)
    OR (kind IN ('income', 'opening') AND amount_minor > 0)
    OR kind = 'transfer'
);

-- Счета вне бюджета при откате становятся наличными.
UPDATE accounts SET kind = 'cash' WHERE kind IN ('wallet', 'piggy');
ALTER TABLE accounts DROP COLUMN off_budget;
ALTER TABLE accounts DROP CONSTRAINT accounts_kind_check;
ALTER TABLE accounts ADD CONSTRAINT accounts_kind_check CHECK (kind IN ('cash', 'debit', 'credit'));
