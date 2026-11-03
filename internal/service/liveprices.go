package service

import (
	"context"
	"math/big"
	"sync"
	"time"

	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// livePriceTTL: как долго цена монеты или акции в портфеле считается свежей. Фиат остаётся дневным.
const livePriceTTL = 10 * time.Minute

type livePrice struct {
	usd *big.Rat
	at  time.Time
}

// livePrices хранит последние цены активов в долларах в памяти, в БД они не попадают.
type livePrices struct {
	mu sync.Mutex
	m  map[money.Currency]livePrice
}

// applyLivePrices заменяет в курсах сегодняшнего дня цены монет и акций на последние рыночные. Курс дня в БД не меняется,
// при ошибке источника остаётся курс дня.
func (s *Service) applyLivePrices(ctx context.Context, book rateBook, today apptime.Day, coins []money.Currency) {
	day := book[today]
	if day == nil {
		return
	}
	for _, c := range coins {
		if c == money.USDT {
			continue // USDT выводится из EUR и не торгуется парой сам к себе
		}
		quote := day[money.USDT]
		if money.IsStock(c) {
			quote = day[money.USD]
		}
		if quote == nil {
			continue
		}
		usd, err := s.livePrice(ctx, c)
		if err != nil {
			s.log.Warn("live price", "coin", string(c), "err", err)
			continue
		}
		day[c] = new(big.Rat).Mul(usd, quote)
	}
}

func (s *Service) livePrice(ctx context.Context, c money.Currency) (*big.Rat, error) {
	s.live.mu.Lock()
	defer s.live.mu.Unlock()
	if p, ok := s.live.m[c]; ok && s.now().Sub(p.at) < livePriceTTL {
		return p.usd, nil
	}
	usd, err := s.fx.AssetPrice(ctx, c, money.IsStock(c))
	if err != nil {
		return nil, err
	}
	if s.live.m == nil {
		s.live.m = map[money.Currency]livePrice{}
	}
	s.live.m[c] = livePrice{usd: usd, at: s.now()}
	return usd, nil
}
