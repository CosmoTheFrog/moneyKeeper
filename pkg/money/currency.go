// Package money хранит коды валют. Суммы в проекте всегда целые, в минорных единицах, с кодом валюты рядом.
package money

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"sync"
)

type Currency string

const (
	GEL  Currency = "GEL"
	EUR  Currency = "EUR"
	USD  Currency = "USD"
	RUB  Currency = "RUB"
	USDT Currency = "USDT"
)

var (
	ErrUnknownCurrency = errors.New("unknown currency")
	ErrBadCode         = errors.New("bad asset code")
	ErrAssetExists     = errors.New("currency already registered")
)

// MaxUnits — потолок точности валюты.
const MaxUnits = 9

// mu защищает registry: пользовательские активы добавляются, пока сервер работает.
var mu sync.RWMutex

// Info описывает валюту: сколько знаков после запятой хранится и берётся ли её курс отдельно по требованию.
// Точность монет выбрана так, чтобы предельная сумма в 10^12 минорных единиц покрывала любые разумные суммы
// (10 000 BTC, сто миллионов DOGE). Менять точность уже заведённой валюты нельзя: хранимые суммы поплывут.
type Info struct {
	Code  Currency
	Units int
	// Coin: курс запрашивается отдельно, только когда валюта есть в записях или остатках. Это монеты (Binance) и акции.
	Coin bool
	// Stock: ценная бумага; цена в долларах США с биржи, на одну акцию, точность 4 знака для дробных долей.
	Stock bool
}

// registry в порядке показа: фиат, USDT, затем монеты примерно по размеру рынка.
var registry = []Info{
	{Code: GEL, Units: 2},
	{Code: EUR, Units: 2},
	{Code: USD, Units: 2},
	{Code: RUB, Units: 2},
	{Code: USDT, Units: 2},
	{Code: "BTC", Units: 8, Coin: true},
	{Code: "ETH", Units: 8, Coin: true},
	{Code: "XRP", Units: 4, Coin: true},
	{Code: "BNB", Units: 6, Coin: true},
	{Code: "SOL", Units: 6, Coin: true},
	{Code: "USDC", Units: 2, Coin: true},
	{Code: "DOGE", Units: 4, Coin: true},
	{Code: "TRX", Units: 4, Coin: true},
	{Code: "ADA", Units: 4, Coin: true},
	{Code: "LINK", Units: 6, Coin: true},
	{Code: "AVAX", Units: 6, Coin: true},
	{Code: "XLM", Units: 4, Coin: true},
	{Code: "BCH", Units: 6, Coin: true},
	{Code: "TON", Units: 6, Coin: true},
	{Code: "LTC", Units: 6, Coin: true},
	{Code: "DOT", Units: 6, Coin: true},
	{Code: "SHIB", Units: 0, Coin: true},
	{Code: "UNI", Units: 6, Coin: true},
	{Code: "NEAR", Units: 6, Coin: true},
	{Code: "ATOM", Units: 6, Coin: true},
	{Code: "AAPL", Units: 4, Coin: true, Stock: true},
	{Code: "MSFT", Units: 4, Coin: true, Stock: true},
	{Code: "GOOGL", Units: 4, Coin: true, Stock: true},
	{Code: "AMZN", Units: 4, Coin: true, Stock: true},
	{Code: "NVDA", Units: 4, Coin: true, Stock: true},
	{Code: "TSLA", Units: 4, Coin: true, Stock: true},
	{Code: "META", Units: 4, Coin: true, Stock: true},
	{Code: "NFLX", Units: 4, Coin: true, Stock: true},
	{Code: "AMD", Units: 4, Coin: true, Stock: true},
	{Code: "DIS", Units: 4, Coin: true, Stock: true},
}

// Base — валюты, которые можно выбрать базовой при старте.
var Base = []Currency{GEL, EUR, USD, RUB}

// Core — валюты, курсы которых тянутся всегда: всё, кроме лари (он опорный) и монет.
var Core = codes(slices.DeleteFunc(slices.Clone(registry), func(i Info) bool { return i.Code == GEL || i.Coin }))

// All — все поддерживаемые валюты счетов в порядке показа: встроенные, затем добавленные пользователями.
func All() []Currency {
	mu.RLock()
	defer mu.RUnlock()
	return codes(registry)
}

// Infos возвращает описание всех валют в порядке показа.
func Infos() []Info {
	mu.RLock()
	defer mu.RUnlock()
	return slices.Clone(registry)
}

var assetCode = regexp.MustCompile(`^[A-Z0-9]{1,10}$`)

// CheckAsset проверяет описание актива: код из заглавных латинских букв и цифр, курс по требованию, точность в пределах.
func CheckAsset(i Info) error {
	if !assetCode.MatchString(string(i.Code)) || !i.Coin || i.Units < 0 || i.Units > MaxUnits {
		return fmt.Errorf("%q: %w", i.Code, ErrBadCode)
	}
	return nil
}

// Lookup возвращает описание валюты из реестра.
func Lookup(c Currency) (Info, bool) { return lookup(c) }

// Register добавляет актив (монету или акцию) в реестр. Встроенные и уже добавленные коды не заменяются.
func Register(i Info) error {
	if err := CheckAsset(i); err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	if slices.ContainsFunc(registry, func(r Info) bool { return r.Code == i.Code }) {
		return fmt.Errorf("%q: %w", i.Code, ErrAssetExists)
	}
	registry = append(registry, i)
	return nil
}

func lookup(c Currency) (Info, bool) {
	mu.RLock()
	defer mu.RUnlock()
	i := slices.IndexFunc(registry, func(i Info) bool { return i.Code == c })
	if i < 0 {
		return Info{}, false
	}
	return registry[i], true
}

// MinorUnits — сколько знаков после запятой у валюты. Для неизвестной два.
func MinorUnits(c Currency) int {
	if i, ok := lookup(c); ok {
		return i.Units
	}
	return 2
}

// IsCoin: курс валюты запрашивается отдельно по требованию (монеты и акции).
func IsCoin(c Currency) bool {
	i, ok := lookup(c)
	return ok && i.Coin
}

// IsStock: ценная бумага, её цена берётся с биржи в долларах.
func IsStock(c Currency) bool {
	i, ok := lookup(c)
	return ok && i.Stock
}

func Parse(code string) (Currency, error) {
	c := Currency(code)
	if _, ok := lookup(c); !ok {
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

func codes(infos []Info) []Currency {
	out := make([]Currency, len(infos))
	for i, in := range infos {
		out[i] = in.Code
	}
	return out
}
