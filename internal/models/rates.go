package models

import (
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// RateQuote: сколько единиц валюты In стоит одна целая единица Currency. Rate это десятичная запись без округления
// до копеек, потому что курс монеты бывает меньше копейки.
type RateQuote struct {
	Currency money.Currency
	Rate     string
}

// RateTable это курсы всех валют на день к валюте In. Валют, на которые курса нет, в таблице нет.
type RateTable struct {
	Day    apptime.Day
	In     money.Currency
	Quotes []RateQuote
}
