import { createContext } from 'react'

const KEY = 'mk.currencies'
const COINS_KEY = 'mk.coinsInRates'

// Монеты и акции в окне «Валюта и курсы» включаются в «Ещё»; по умолчанию там только обычные валюты.
export const CoinsInRatesContext = createContext(false)

export function readCoinsInRates(): boolean {
  try {
    return localStorage.getItem(COINS_KEY) === '1'
  } catch {
    return false // хранилище может быть недоступно
  }
}

export function saveCoinsInRates(on: boolean): void {
  try {
    localStorage.setItem(COINS_KEY, on ? '1' : '0')
  } catch {
    // не сохранилось, выбор живёт до закрытия
  }
}

// readEnabled: валюты, которые пользователь включил в «Ещё → Мои валюты»; null, пока не выбирал.
export function readEnabled(): string[] | null {
  try {
    const raw = localStorage.getItem(KEY)
    const list: unknown = raw === null ? null : JSON.parse(raw)
    return Array.isArray(list) ? list.filter((c): c is string => typeof c === 'string') : null
  } catch {
    return null // хранилище может быть недоступно
  }
}

export function saveEnabled(list: string[]): void {
  try {
    localStorage.setItem(KEY, JSON.stringify(list))
  } catch {
    // не сохранилось, выбор живёт до закрытия
  }
}

// visibleCurrencies: валюты для выбора в формах. Пока выбора нет, это обычные валюты без монет и акций.
// Базовая валюта и валюты уже заведённых счетов (always) видны всегда, иначе старые записи не к чему привязать.
export function visibleCurrencies(all: string[], enabled: string[] | null, always: string[], isPlain: (code: string) => boolean): string[] {
  return all.filter((c) => always.includes(c) || (enabled === null ? isPlain(c) : enabled.includes(c)))
}
