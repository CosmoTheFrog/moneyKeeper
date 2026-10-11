// Package quickinput разбирает быстрый ввод вида «кофе 5 EUR» на сумму, валюту и заметку.
package quickinput

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"moneykeeper/pkg/money"
)

var (
	ErrNoAmount      = errors.New("no amount in text")
	ErrInvalidAmount = errors.New("invalid amount")
)

const maxMajor = 10_000_000_000

// Result: Currency пустая, если в тексте валюты нет, её подставляет вызывающий.
type Result struct {
	AmountMinor int64
	Currency    money.Currency
	Note        string
}

var (
	amountRe = regexp.MustCompile(`^(\p{Sc})?(\d+(?:[.,]\d+)?)(\p{Sc}|[A-Za-z]{3,4})?$`)
	symbols  = map[string]money.Currency{"₾": money.GEL, "€": money.EUR, "$": money.USD, "₽": money.RUB}
)

type candidate struct {
	index    int
	number   string
	currency money.Currency
	nextUsed bool // валюта стоит отдельным словом после числа
}

// Parse берёт первое число с валютой, а если валюты нигде нет, первое число.
func Parse(text string) (Result, error) {
	tokens := strings.Fields(text)

	var first, withCurrency *candidate
	for i, tok := range tokens {
		m := amountRe.FindStringSubmatch(tok)
		if m == nil {
			continue
		}
		cur, ok := currencyOf(m[1], m[3])
		if !ok {
			continue
		}
		c := &candidate{index: i, number: m[2], currency: cur}
		if cur == "" && i+1 < len(tokens) {
			if next, ok := currencyWord(tokens[i+1]); ok {
				c.currency, c.nextUsed = next, true
			}
		}
		if first == nil {
			first = c
		}
		if c.currency != "" && withCurrency == nil {
			withCurrency = c
		}
	}
	picked := withCurrency
	if picked == nil {
		picked = first
	}
	if picked == nil {
		return Result{}, ErrNoAmount
	}

	amount, err := toMinor(picked.number, money.MinorUnits(money.GEL))
	if err != nil {
		return Result{}, err
	}
	note := make([]string, 0, len(tokens))
	for j, t := range tokens {
		if j == picked.index || (picked.nextUsed && j == picked.index+1) {
			continue
		}
		note = append(note, t)
	}
	return Result{AmountMinor: amount, Currency: picked.currency, Note: strings.Join(note, " ")}, nil
}

// currencyOf берёт валюту из символа перед числом или из суффикса. Пусто и ok, если валюты в токене нет.
func currencyOf(prefix, suffix string) (money.Currency, bool) {
	for _, s := range []string{prefix, suffix} {
		if s == "" {
			continue
		}
		c, ok := currencyWord(s)
		return c, ok
	}
	return "", true
}

func currencyWord(w string) (money.Currency, bool) {
	if c, ok := symbols[w]; ok {
		return c, true
	}
	c, err := money.Parse(strings.ToUpper(w))
	return c, err == nil
}

func toMinor(s string, units int) (int64, error) {
	whole, frac, _ := strings.Cut(strings.Replace(s, ",", ".", 1), ".")
	if len(frac) > units {
		return 0, fmt.Errorf("%q: %w", s, ErrInvalidAmount)
	}
	frac += strings.Repeat("0", units-len(frac))

	var major, minor int64
	for _, r := range whole {
		major = major*10 + int64(r-'0')
		if major > maxMajor {
			return 0, fmt.Errorf("%q: %w", s, ErrInvalidAmount)
		}
	}
	for _, r := range frac {
		minor = minor*10 + int64(r-'0')
	}
	total := major
	for range units {
		total *= 10
	}
	total += minor
	if total == 0 {
		return 0, ErrNoAmount
	}
	return total, nil
}
