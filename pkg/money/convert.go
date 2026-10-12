package money

import (
	"errors"
	"fmt"
	"math/big"
)

var ErrBadRate = errors.New("bad rate")

// Convert пересчитывает сумму в минорных единицах из from в to через GEL: gelPerFrom и gelPerTo это цена
// одной целой единицы валюты в лари (для GEL единица). Вычисления точные (big.Rat), округление
// один раз в конце, половина от нуля. Деньги остаются целыми, дробей и float тут нет.
func Convert(amountMinor int64, from, to Currency, gelPerFrom, gelPerTo *big.Rat) (int64, error) {
	if gelPerFrom == nil || gelPerTo == nil || gelPerFrom.Sign() <= 0 || gelPerTo.Sign() <= 0 {
		return 0, fmt.Errorf("%s to %s: %w", from, to, ErrBadRate)
	}
	r := new(big.Rat).SetInt64(amountMinor)
	r.Mul(r, gelPerFrom)
	r.Quo(r, gelPerTo)

	// Разные точности валют: 10^(знаки to) / 10^(знаки from).
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(abs(MinorUnits(to)-MinorUnits(from)))), nil)
	if MinorUnits(to) >= MinorUnits(from) {
		r.Mul(r, new(big.Rat).SetInt(scale))
	} else {
		r.Quo(r, new(big.Rat).SetInt(scale))
	}

	out, ok := roundHalfAway(r)
	if !ok {
		return 0, fmt.Errorf("%d %s to %s overflows: %w", amountMinor, from, to, ErrBadRate)
	}
	return out, nil
}

func roundHalfAway(r *big.Rat) (int64, bool) {
	num := new(big.Int).Abs(r.Num())
	q, rem := new(big.Int).QuoRem(num, r.Denom(), new(big.Int))
	if rem.Lsh(rem, 1).Cmp(r.Denom()) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	if r.Sign() < 0 {
		q.Neg(q)
	}
	if !q.IsInt64() {
		return 0, false
	}
	return q.Int64(), true
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
