// Суммы везде целые, в минорных единицах (копейках). Дробей в расчётах нет, только целочисленный разбор.
const SYMBOLS: Record<string, string> = { GEL: '₾', EUR: '€', USD: '$', RUB: '₽', USDT: 'USDT' }

const MINUS = '−'
const NBSP = ' '

export function currencySign(currency: string): string {
  return SYMBOLS[currency] ?? currency
}

function groupDigits(whole: number): string {
  return String(whole).replace(/\B(?=(\d{3})+(?!\d))/g, NBSP)
}

// formatMinor(-123450, 'GEL') → «−1 234,50 ₾»; signed добавляет «+» к положительным.
export function formatMinor(minor: number, currency: string, signed = false): string {
  const abs = Math.abs(minor)
  const whole = Math.trunc(abs / 100)
  const frac = String(abs % 100).padStart(2, '0')
  const sign = minor < 0 ? MINUS : signed && minor > 0 ? '+' : ''
  return `${sign}${groupDigits(whole)},${frac}${NBSP}${currencySign(currency)}`
}

const MAX_MINOR = 1_000_000_000_000

// parseAmount разбирает ввод вроде «12», «12,5», «12.50» в минорные единицы. Ноль и мусор дают null.
export function parseAmount(input: string): number | null {
  const m = /^(\d+)(?:[.,](\d{1,2}))?$/.exec(input.trim().replace(/\s/g, ''))
  if (m === null) return null
  const minor = Number(m[1]) * 100 + Number((m[2] ?? '').padEnd(2, '0'))
  return minor > 0 && minor <= MAX_MINOR ? minor : null
}

// formatInput возвращает сумму в виде для поля ввода: 123450 → «1234,50».
export function formatInput(minor: number): string {
  const abs = Math.abs(minor)
  return `${Math.trunc(abs / 100)},${String(abs % 100).padStart(2, '0')}`
}
