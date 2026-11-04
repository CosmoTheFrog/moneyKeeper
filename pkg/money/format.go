package money

import (
	"fmt"
	"strings"
)

var symbols = map[Currency]string{GEL: "₾", EUR: "€", USD: "$", RUB: "₽"}

const nbsp = "\u00a0"

// Decimal записывает сумму для машин: «-1234.50», точка, без разделителей и знака валюты. Для выгрузки в CSV.
func Decimal(minor int64, c Currency) string {
	sign := ""
	if minor < 0 {
		sign, minor = "-", -minor
	}
	units := MinorUnits(c)
	if units == 0 {
		return fmt.Sprintf("%s%d", sign, minor)
	}
	pow := int64(1)
	for range units {
		pow *= 10
	}
	return fmt.Sprintf("%s%d.%0*d", sign, minor/pow, units, minor%pow)
}

// Format записывает сумму для людей: «1 234,50 ₾», знаков после запятой столько, сколько у валюты. Для текста
// сообщений бота; в интерфейсе суммы показывает единый компонент.
func Format(minor int64, c Currency) string {
	sign := ""
	if minor < 0 {
		sign, minor = "−", -minor
	}
	units := MinorUnits(c)
	pow := int64(1)
	for range units {
		pow *= 10
	}
	whole, frac := minor/pow, minor%pow

	digits := fmt.Sprint(whole)
	var b strings.Builder
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteString(nbsp)
		}
		b.WriteRune(r)
	}
	out := sign + b.String()
	if units > 0 {
		out += fmt.Sprintf(",%0*d", units, frac)
	}
	name := string(c)
	if s, ok := symbols[c]; ok {
		name = s
	}
	return out + nbsp + name
}
