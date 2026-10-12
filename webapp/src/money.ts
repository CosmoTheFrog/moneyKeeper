// Суммы везде целые, в минорных единицах (копейках, сатоши). Дробей в расчётах нет, только целочисленный разбор.
const SYMBOLS: Record<string, string> = { GEL: '₾', EUR: '€', USD: '$', RUB: '₽', USDT: 'USDT' }

const MINUS = '−'
const NBSP = ' '
const DEFAULT_UNITS = 2

// Сколько знаков после запятой у каждой валюты и какие из них монеты. Приходит с сервера при входе (/api/me),
// до этого у всех валют два знака.
let units: Record<string, number> = {}
let coins = new Set<string>()

export type CurrencyInfo = { code: string; units: number; coin: boolean }

export function registerCurrencies(infos: CurrencyInfo[]): void {
  units = Object.fromEntries(infos.map((i) => [i.code, i.units]))
  coins = new Set(infos.filter((i) => i.coin).map((i) => i.code))
}

export function unitsOf(currency: string): number {
  return units[currency] ?? DEFAULT_UNITS
}

export function isCoin(currency: string): boolean {
  return coins.has(currency)
}

export function currencySign(currency: string): string {
  return SYMBOLS[currency] ?? currency
}

function groupDigits(whole: number): string {
  return String(whole).replace(/\B(?=(\d{3})+(?!\d))/g, NBSP)
}

function pow10(n: number): number {
  return 10 ** n
}

// formatMinor(-123450, 'GEL') → «−1 234,50 ₾»; signed добавляет «+» к положительным. Знаков столько, сколько у валюты.
export function formatMinor(minor: number, currency: string, signed = false): string {
  const u = unitsOf(currency)
  const abs = Math.abs(minor)
  const whole = Math.trunc(abs / pow10(u))
  const frac = u === 0 ? '' : `,${String(abs % pow10(u)).padStart(u, '0')}`
  const sign = minor < 0 ? MINUS : signed && minor > 0 ? '+' : ''
  return `${sign}${groupDigits(whole)}${frac}${NBSP}${currencySign(currency)}`
}

const MAX_MINOR = 1_000_000_000_000

// parseAmount разбирает ввод вроде «12», «12,5», «12.50» в минорные единицы валюты currency. Ноль и мусор дают null,
// как и лишние знаки после запятой.
export function parseAmount(input: string, currency: string): number | null {
  const u = unitsOf(currency)
  const m = new RegExp(`^(\\d+)(?:[.,](\\d{1,${Math.max(u, 1)}}))?$`).exec(input.trim().replace(/\s/g, ''))
  if (m === null || (u === 0 && m[2] !== undefined)) return null
  const minor = Number(m[1]) * pow10(u) + Number((m[2] ?? '').padEnd(u, '0') || 0)
  return minor > 0 && minor <= MAX_MINOR ? minor : null
}

// formatInput возвращает сумму в виде для поля ввода: 123450 в GEL → «1234,50».
export function formatInput(minor: number, currency: string): string {
  const u = unitsOf(currency)
  const abs = Math.abs(minor)
  const whole = Math.trunc(abs / pow10(u))
  return u === 0 ? String(whole) : `${whole},${String(abs % pow10(u)).padStart(u, '0')}`
}
