package service

import (
	"context"
	"math/big"
	"slices"
	"sort"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// holding это позиция по монете: сколько штук, сколько за них заплачено и сколько уже заработано продажами.
// Всё в лари, потому что курсы хранятся к лари.
type holding struct {
	qty         int64
	costGEL     int64
	realizedGEL int64
}

// valuer оценивает сумму в лари по курсу дня. false, если курса на этот день нет.
type valuer func(amount int64, cur money.Currency, day apptime.Day) (int64, bool)

// buildHoldings считает позиции по монетам по среднему: покупка добавляет штуки и цену, продажа убирает штуки вместе с
// пропорциональной частью цены и фиксирует разницу с выручкой. Комиссия сети и комиссия в монете уменьшают штуки,
// не трогая цену, поэтому средняя цена растёт. Покупка и продажа оцениваются по второй стороне обмена, остальные
// поступления монет (начальный остаток, доход) по рыночному курсу дня. missing считает суммы без курса.
func buildHoldings(txs []models.Transaction, toGEL valuer) (map[money.Currency]*holding, int) {
	out := map[money.Currency]*holding{}
	missing := 0
	hold := func(c money.Currency) *holding {
		if out[c] == nil {
			out[c] = &holding{}
		}
		return out[c]
	}
	value := func(amount int64, cur money.Currency, day apptime.Day) int64 {
		if amount < 0 {
			amount = -amount
		}
		v, ok := toGEL(amount, cur, day)
		if !ok {
			missing++
		}
		return v
	}
	buy := func(c money.Currency, qty, cost int64) {
		h := hold(c)
		h.qty += qty
		h.costGEL += cost
	}
	sell := func(c money.Currency, qty, proceeds int64) {
		h := hold(c)
		costOut := int64(0)
		if h.qty > 0 {
			costOut = proportion(h.costGEL, min(qty, h.qty), h.qty)
		}
		h.qty -= qty
		h.costGEL -= costOut
		h.realizedGEL += proceeds - costOut
	}
	lose := func(c money.Currency, qty int64) { hold(c).qty -= qty }

	// Группы (обмен, перевод с комиссией) обрабатываются целиком, по порядку первой записи.
	type group struct {
		rows []models.Transaction
	}
	var order []*group
	byID := map[int64]*group{}
	for _, t := range txs {
		if t.TransferID == nil {
			order = append(order, &group{rows: []models.Transaction{t}})
			continue
		}
		g := byID[*t.TransferID]
		if g == nil {
			g = &group{}
			byID[*t.TransferID] = g
			order = append(order, g)
		}
		g.rows = append(g.rows, t)
	}

	for _, g := range order {
		first := g.rows[0]
		switch {
		case slices.ContainsFunc(g.rows, func(t models.Transaction) bool { return t.Kind == models.KindExchange }):
			var gave, got *models.Transaction
			var feeGEL int64
			var coinFees []models.Transaction // комиссии в монете списываются после сделки, чтобы не искажать цену продажи
			for i := range g.rows {
				t := g.rows[i]
				switch {
				case t.Kind == models.KindExchange && t.AmountMinor < 0:
					gave = &g.rows[i]
				case t.Kind == models.KindExchange:
					got = &g.rows[i]
				case money.IsCoin(t.Currency):
					coinFees = append(coinFees, t)
				default:
					feeGEL += value(t.AmountMinor, t.Currency, t.OccurredOn)
				}
			}
			if gave == nil || got == nil {
				continue
			}
			paid, received := value(gave.AmountMinor, gave.Currency, first.OccurredOn), value(got.AmountMinor, got.Currency, first.OccurredOn)
			switch {
			case money.IsCoin(gave.Currency) && money.IsCoin(got.Currency):
				sell(gave.Currency, -gave.AmountMinor, paid)
				buy(got.Currency, got.AmountMinor, paid+feeGEL)
			case money.IsCoin(got.Currency):
				buy(got.Currency, got.AmountMinor, paid+feeGEL)
			case money.IsCoin(gave.Currency):
				sell(gave.Currency, -gave.AmountMinor, received-feeGEL)
			}
			for _, f := range coinFees {
				lose(f.Currency, -f.AmountMinor)
			}
		case first.Kind == models.KindTransfer:
			// Перевод монеты между кошельками: потеряна только разница, то есть комиссия сети.
			var sum int64
			for _, t := range g.rows {
				sum += t.AmountMinor
			}
			if money.IsCoin(first.Currency) && sum < 0 {
				lose(first.Currency, -sum)
			}
		default:
			for _, t := range g.rows {
				if !money.IsCoin(t.Currency) {
					continue
				}
				switch {
				case t.Kind == models.KindFee:
					lose(t.Currency, -t.AmountMinor)
				case t.AmountMinor > 0:
					buy(t.Currency, t.AmountMinor, value(t.AmountMinor, t.Currency, t.OccurredOn))
				default:
					sell(t.Currency, -t.AmountMinor, value(t.AmountMinor, t.Currency, t.OccurredOn))
				}
			}
		}
	}
	return out, missing
}

// proportion считает part*total/whole с округлением до ближайшего целого, без переполнения.
func proportion(total, part, whole int64) int64 {
	r := new(big.Rat).SetFrac(new(big.Int).Mul(big.NewInt(total), big.NewInt(part)), big.NewInt(whole))
	num, den := r.Num(), r.Denom()
	q, rem := new(big.Int).QuoRem(num, den, new(big.Int))
	if new(big.Int).Lsh(rem.Abs(rem), 1).Cmp(den) >= 0 {
		if num.Sign() >= 0 {
			q.Add(q, big.NewInt(1))
		} else {
			q.Sub(q, big.NewInt(1))
		}
	}
	return q.Int64()
}

// Portfolio собирает позиции по монетам во всех кошельках в валюте in (пусто: базовая валюта бюджета, иначе лари).
// Текущая цена по сегодняшнему курсу, цена покупки по курсу дня каждой покупки, пересчитанная в валюту портфеля по сегодняшнему.
func (s *Service) Portfolio(ctx context.Context, u models.User, in string) (models.Portfolio, error) {
	target, ok, err := s.target(ctx, u, in)
	if err != nil {
		return models.Portfolio{}, err
	}
	if !ok {
		target = money.GEL
	}
	var coins []string
	var coinList []money.Currency
	for _, c := range money.All {
		if money.IsCoin(c) {
			coins = append(coins, string(c))
			coinList = append(coinList, c)
		}
	}
	txs, err := s.transactions.CoinActivity(ctx, u.ActiveBudgetID, coins)
	if err != nil {
		return models.Portfolio{}, err
	}

	today := s.today(u)
	days := []apptime.Day{today}
	used := map[money.Currency]bool{target: true}
	for _, t := range txs {
		days = append(days, s.rateDay(u, t.OccurredOn))
		used[t.Currency] = true
	}
	var wanted []money.Currency
	for c := range used {
		wanted = append(wanted, c)
	}
	book := s.rates(ctx, days, onlyCoins(wanted))

	holdings, missing := buildHoldings(txs, func(amount int64, cur money.Currency, day apptime.Day) (int64, bool) {
		return book.convert(amount, cur, money.GEL, s.rateDay(u, day))
	})

	out := models.Portfolio{Currency: target, MissingRates: missing}
	for _, c := range coinList {
		h := holdings[c]
		if h == nil || (h.qty <= 0 && h.realizedGEL == 0) {
			continue
		}
		item := models.PortfolioItem{Currency: c, QuantityMinor: max(h.qty, 0)}
		inTarget := func(gel int64) int64 {
			v, _ := book.convert(gel, money.GEL, target, today)
			return v
		}
		if item.QuantityMinor > 0 {
			if v, ok := book.convert(item.QuantityMinor, c, target, today); ok {
				item.ValueMinor = v
			} else {
				out.MissingRates++
			}
			item.CostMinor = inTarget(h.costGEL)
			item.ProfitMinor = item.ValueMinor - item.CostMinor
			if item.CostMinor > 0 {
				bp := proportion(item.ProfitMinor, 10000, item.CostMinor)
				item.ProfitBP = &bp
				item.AvgPrice = avgPrice(item.CostMinor, item.QuantityMinor, target, c)
			}
		}
		item.RealizedMinor = inTarget(h.realizedGEL)
		out.Items = append(out.Items, item)
	}
	sort.SliceStable(out.Items, func(i, j int) bool { return out.Items[i].ValueMinor > out.Items[j].ValueMinor })
	return out, nil
}

// avgPrice это цена за целую монету: стоимость покупки в минорных единицах валюты портфеля на число монет.
func avgPrice(costMinor, qtyMinor int64, target, coin money.Currency) string {
	r := new(big.Rat).SetFrac(big.NewInt(costMinor), pow10(money.MinorUnits(target)))
	r.Quo(r, new(big.Rat).SetFrac(big.NewInt(qtyMinor), pow10(money.MinorUnits(coin))))
	return trimRate(r)
}

func pow10(n int) *big.Int { return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil) }
