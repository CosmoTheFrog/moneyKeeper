-- Режим интерфейса: simple (траты по категориям, один скрытый счёт) или full (карты, переводы, обмен, копилка).
-- NULL значит, что пользователь ещё не выбирал: приложение покажет выбор при входе. Существующие бюджеты остаются полными.
ALTER TABLE budgets ADD COLUMN interface_mode TEXT CHECK (interface_mode IN ('simple', 'full'));
UPDATE budgets SET interface_mode = 'full';
