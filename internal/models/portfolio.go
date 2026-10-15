package models

import "moneykeeper/pkg/money"

// PortfolioItem это позиция по одной монете во всех кошельках. Суммы в валюте портфеля. CostMinor это сколько стоили
// монеты при покупке (по курсу дня сделки), ProfitMinor разница с текущей ценой. ProfitBP прибыль в сотых долях
// процента, nil если цена покупки неизвестна. RealizedMinor прибыль, уже зафиксированная продажами.
type PortfolioItem struct {
	Currency      money.Currency
	QuantityMinor int64
	ValueMinor    int64
	CostMinor     int64
	ProfitMinor   int64
	ProfitBP      *int64
	RealizedMinor int64
	// AvgPrice это средняя цена покупки за целую монету в валюте портфеля, десятичной строкой. Пусто без цены покупки.
	AvgPrice string
}

type Portfolio struct {
	Currency money.Currency
	Items    []PortfolioItem
	// MissingRates число записей, которые не удалось оценить из-за отсутствия курса: они не вошли в стоимость.
	MissingRates int
}
