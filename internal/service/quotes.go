package service

import (
	"context"
	"math/big"
	"strings"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// rateDigits: знаков после запятой в записи курса, хватает на SHIB и подобные монеты дешевле копейки.
const rateDigits = 12

// Rates отдаёт курсы всех валют к валюте in на сегодня. Пусто: базовая валюта бюджета, а если её ещё нет, лари.
// Курс дня сохраняется при первом запросе и до конца суток не меняется.
func (s *Service) Rates(ctx context.Context, u models.User, in string) (models.RateTable, error) {
	target, ok, err := s.target(ctx, u, in)
	if err != nil {
		return models.RateTable{}, err
	}
	if !ok {
		target = money.GEL
	}

	today := s.today(u)
	var coins []money.Currency
	for _, c := range money.All {
		if money.IsCoin(c) {
			coins = append(coins, c)
		}
	}
	book := s.rates(ctx, []apptime.Day{today}, coins)

	table := models.RateTable{Day: today, In: target}
	gelTo, ok := book.gel(target, today)
	if !ok {
		return table, nil
	}
	for _, c := range money.All {
		gelFrom, ok := book.gel(c, today)
		if c == target || !ok {
			continue
		}
		table.Quotes = append(table.Quotes, models.RateQuote{Currency: c, Rate: trimRate(new(big.Rat).Quo(gelFrom, gelTo))})
	}
	return table, nil
}

func trimRate(r *big.Rat) string {
	s := strings.TrimRight(r.FloatString(rateDigits), "0")
	return strings.TrimSuffix(s, ".")
}
