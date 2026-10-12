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

// formatRate показывает курс из десятичной строки сервера: без float, с округлением до значимых цифр.
// Курс от 100 с двумя знаками, от 1 с четырьмя, меньше единицы четыре значащих цифры после нулей.
export function formatRate(rate: string): string {
  const [whole = '0', frac = ''] = rate.split('.')
  const digits = 12
  const padded = frac.padEnd(digits, '0').slice(0, digits)
  const leadingZeros = whole === '0' ? (/^0*/.exec(padded)?.[0].length ?? 0) : 0
  const keep = whole === '0' ? Math.min(leadingZeros + 4, digits) : whole.length >= 3 ? 2 : 4
  const scale = 10n ** BigInt(digits - keep)
  const all = BigInt(whole + padded)
  let q = all / scale
  if ((all % scale) * 2n >= scale) q += 1n
  const text = q.toString().padStart(keep + 1, '0')
  const intPart = text.slice(0, text.length - keep)
  const fracPart = text.slice(text.length - keep).replace(/0+$/, '')
  return `${groupDigits(Number(intPart))}${fracPart === '' ? '' : `,${fracPart}`}`
}

// formatInput возвращает сумму в виде для поля ввода: 123450 в GEL → «1234,50».
export function formatInput(minor: number, currency: string): string {
  const u = unitsOf(currency)
  const abs = Math.abs(minor)
  const whole = Math.trunc(abs / pow10(u))
  return u === 0 ? String(whole) : `${whole},${String(abs % pow10(u)).padStart(u, '0')}`
}
