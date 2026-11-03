package models

import (
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// ImportRow это строка выписки, приведённая к единому виду. AmountMinor со знаком: расход отрицательный.
type ImportRow struct {
	Date        apptime.Day
	AmountMinor int64
	Currency    money.Currency
	Description string
	// CategoryID: у расхода категория обязательна при сохранении, у дохода её нет.
	CategoryID *int64
	// SourceCategory: название категории в файле, из которого строка пришла. Заполнено, если точного совпадения с
	// категориями бюджета нет и пользователю нужно выбрать категорию самому.
	SourceCategory string
	// Suggested: категорию предложила модель, пользователь её ещё не подтвердил.
	Suggested bool
	// Duplicate: на счёте уже есть запись с той же датой, суммой и валютой.
	Duplicate bool
}

// ImportPreview это разобранная выписка до сохранения. Skipped это число строк, которые не прошли проверку.
type ImportPreview struct {
	Rows    []ImportRow
	Skipped int
}
