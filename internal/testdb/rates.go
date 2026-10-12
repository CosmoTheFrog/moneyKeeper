package testdb

import (
	"context"
	"errors"
	"math/big"
	"sync"

	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// Rates подменяет источник курсов: на любой день одни и те же курсы, число обращений считается.
// С заданным Err отдаёт только эту ошибку, как недоступный источник.
type Rates struct {
	Err error

	mu    sync.Mutex
	calls int
}

// Fixed это курсы, которые отдаёт Rates: цена одной целой единицы в GEL.
var Fixed = map[money.Currency]string{
	money.USD:  "2.7084",
	money.EUR:  "3.1594",
	money.RUB:  "0.0299",
	money.USDT: "2.73",
}

func (r *Rates) GelPerUnit(_ context.Context, _ apptime.Day) (map[money.Currency]*big.Rat, error) {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	if r.Err != nil {
		return nil, r.Err
	}
	out := map[money.Currency]*big.Rat{}
	for c, s := range Fixed {
		v, ok := new(big.Rat).SetString(s)
		if !ok {
			return nil, errors.New("bad fixed rate " + s)
		}
		out[c] = v
	}
	return out, nil
}

func (r *Rates) Calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}
