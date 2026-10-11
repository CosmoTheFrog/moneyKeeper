// Package money хранит коды валют. Суммы в проекте всегда целые, в минорных единицах, с кодом валюты рядом.
package money

import (
	"errors"
	"fmt"
	"slices"
)

type Currency string

const (
	GEL  Currency = "GEL"
	EUR  Currency = "EUR"
	USD  Currency = "USD"
	RUB  Currency = "RUB"
	USDT Currency = "USDT"
)

var ErrUnknownCurrency = errors.New("unknown currency")

// Base — валюты, которые можно выбрать базовой при старте.
var Base = []Currency{GEL, EUR, USD, RUB}

// All — все поддерживаемые валюты счетов.
var All = []Currency{GEL, EUR, USD, RUB, USDT}

// MinorUnits — сколько знаков после запятой у валюты. Пока везде два, у крипты
// точность уточним вместе с курсами.
func MinorUnits(Currency) int { return 2 }

func Parse(code string) (Currency, error) {
	c := Currency(code)
	if !slices.Contains(All, c) {
		return "", fmt.Errorf("%q: %w", code, ErrUnknownCurrency)
	}
	return c, nil
}

func ParseBase(code string) (Currency, error) {
	c, err := Parse(code)
	if err != nil {
		return "", err
	}
	if !slices.Contains(Base, c) {
		return "", fmt.Errorf("%q is not a base currency: %w", code, ErrUnknownCurrency)
	}
	return c, nil
}
