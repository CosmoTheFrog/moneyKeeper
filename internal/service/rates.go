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

// maxRateFetches ограничивает число походов за курсами в одном запросе: после выкладки история
// подтягивается постепенно, а не одним долгим ответом.
const maxRateFetches = 8

// RateSource отдаёт курсы дня: цена одной целой единицы валюты в GEL. Фиат и USDT всегда, монеты из coins
// по требованию. При частичном успехе возвращает полученное вместе с ошибкой.
type RateSource interface {
	GelPerUnit(ctx context.Context, day apptime.Day, coins []money.Currency) (map[money.Currency]*big.Rat, error)
}

// rateBook это курсы на набор дней. Пересчёт идёт только по курсу своего дня, без подмены соседним.
type rateBook map[apptime.Day]map[money.Currency]*big.Rat

func (b rateBook) convert(amountMinor int64, from, to money.Currency, day apptime.Day) (int64, bool) {
	if from == to {
		return amountMinor, true
	}
	gelFrom, okFrom := b.gel(from, day)
	gelTo, okTo := b.gel(to, day)
	if !okFrom || !okTo {
		return 0, false
	}
	out, err := money.Convert(amountMinor, from, to, gelFrom, gelTo)
	return out, err == nil
}

func (b rateBook) gel(c money.Currency, day apptime.Day) (*big.Rat, bool) {
	if c == money.GEL {
		return big.NewRat(1, 1), true
	}
	r, ok := b[day][c]
	return r, ok
}

// rateDay: записи будущим числом пересчитываются по курсу сегодняшнего дня, будущих курсов не бывает.
func (s *Service) rateDay(u models.User, d apptime.Day) apptime.Day {
	if today := s.today(u); today.Before(d) {
		return today
	}
	return d
}

// rates возвращает курсы на дни, докачивая недостающие: основные валюты всегда, монеты из coins только если они
// нужны. Недоступный источник не ошибка: сумм в базовой валюте просто не будет, остальной ответ остаётся рабочим.
func (s *Service) rates(ctx context.Context, days []apptime.Day, coins []money.Currency) rateBook {
	days = uniqueDays(days)
	book, err := s.rateStore.ForDays(ctx, days)
	if err != nil {
		s.log.Warn("load rates", "err", err)
		return rateBook{}
	}

	// Для каждого дня запоминаем, каких монет не хватает: основные валюты проверяются всегда.
	var missing []apptime.Day
	absent := map[apptime.Day][]money.Currency{}
	for _, d := range days {
		lacksCore := slices.ContainsFunc(money.Core, func(c money.Currency) bool { return book[d][c] == nil })
		for _, c := range coins {
			if book[d][c] == nil {
				absent[d] = append(absent[d], c)
			}
		}
		if lacksCore || len(absent[d]) > 0 {
			missing = append(missing, d)
		}
	}
	sort.Slice(missing, func(i, j int) bool { return missing[j].Before(missing[i]) }) // сначала свежие
	if len(missing) > maxRateFetches {
		missing = missing[:maxRateFetches]
	}

	fetched := false
	for _, d := range missing {
		got, err := s.fx.GelPerUnit(ctx, d, absent[d])
		if err != nil {
			s.log.Warn("fetch rates", "day", d.String(), "err", err)
		}
		if len(got) > 0 {
			if err := s.rateStore.Store(ctx, d, got); err != nil {
				s.log.Warn("store rates", "day", d.String(), "err", err)
				return book
			}
			fetched = true
		}
		if err != nil && len(got) == 0 {
			break // источник недоступен: не ждём таймаута на каждый день
		}
	}
	if !fetched {
		return book
	}
	if book, err = s.rateStore.ForDays(ctx, days); err != nil {
		s.log.Warn("reload rates", "err", err)
		return rateBook{}
	}
	return book
}

// baseCurrency возвращает базовую валюту бюджета, false если она ещё не выбрана.
func (s *Service) baseCurrency(ctx context.Context, u models.User) (money.Currency, bool, error) {
	b, err := s.budgets.Get(ctx, u.ActiveBudgetID)
	if err != nil || b.BaseCurrency == nil {
		return "", false, err
	}
	return *b.BaseCurrency, true, nil
}

// target выбирает валюту пересчёта: in, если задана, иначе базовая валюта бюджета. false, если пересчитывать не во что.
func (s *Service) target(ctx context.Context, u models.User, in string) (money.Currency, bool, error) {
	if in == "" {
		return s.baseCurrency(ctx, u)
	}
	c, err := money.Parse(in)
	return c, err == nil, err
}

// priceTransactions проставляет BaseMinor по курсу дня каждой записи в валюте in (пусто: базовая).
func (s *Service) priceTransactions(ctx context.Context, u models.User, ts []models.Transaction, in string) error {
	base, ok, err := s.target(ctx, u, in)
	if err != nil || !ok || len(ts) == 0 {
		return err
	}
	days := make([]apptime.Day, 0, len(ts))
	for _, t := range ts {
		days = append(days, s.rateDay(u, t.OccurredOn))
	}
	coins := make([]money.Currency, 0, len(ts))
	for _, t := range ts {
		coins = append(coins, t.Currency)
	}
	book := s.rates(ctx, days, onlyCoins(append(coins, base)))
	for i, t := range ts {
		if v, ok := book.convert(t.AmountMinor, t.Currency, base, s.rateDay(u, t.OccurredOn)); ok {
			ts[i].BaseMinor = &v
		}
	}
	return nil
}

// priceAccounts проставляет остатки в валюте in (пусто: базовая) по сегодняшнему курсу и расходы по курсу дня каждой записи.
func (s *Service) priceAccounts(ctx context.Context, u models.User, accs []models.Account, in string) error {
	base, ok, err := s.target(ctx, u, in)
	if err != nil || !ok || len(accs) == 0 {
		return err
	}
	spends, err := s.accounts.DailySpends(ctx, u.ActiveBudgetID)
	if err != nil {
		return err
	}
	today := s.today(u)
	days := []apptime.Day{today}
	type key struct {
		account  int64
		currency money.Currency
	}
	spendsOf := map[key][]models.DailySpend{}
	for _, sp := range spends {
		days = append(days, s.rateDay(u, sp.Day))
		k := key{sp.AccountID, sp.Currency}
		spendsOf[k] = append(spendsOf[k], sp)
	}
	var held []money.Currency
	for _, a := range accs {
		for _, b := range a.Balances {
			held = append(held, b.Currency)
		}
	}
	book := s.rates(ctx, days, onlyCoins(append(held, base)))

	for ai := range accs {
		for bi := range accs[ai].Balances {
			b := &accs[ai].Balances[bi]
			if v, ok := book.convert(b.BalanceMinor, b.Currency, base, today); ok {
				b.BaseBalanceMinor = &v
			}
			var total int64
			complete := true
			for _, sp := range spendsOf[key{accs[ai].ID, b.Currency}] {
				v, ok := book.convert(sp.SpentMinor, b.Currency, base, s.rateDay(u, sp.Day))
				if !ok {
					complete = false
					break
				}
				total += v
			}
			if complete {
				b.BaseSpentMinor = &total
			}
		}
	}
	return nil
}

// onlyCoins оставляет монеты без повторов: только их курсы запрашиваются отдельно.
func onlyCoins(cs []money.Currency) []money.Currency {
	var out []money.Currency
	for _, c := range cs {
		if money.IsCoin(c) && !slices.Contains(out, c) {
			out = append(out, c)
		}
	}
	return out
}

func uniqueDays(days []apptime.Day) []apptime.Day {
	seen := map[apptime.Day]bool{}
	out := make([]apptime.Day, 0, len(days))
	for _, d := range days {
		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	return out
}
