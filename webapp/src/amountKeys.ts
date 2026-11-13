// Ввод суммы с экранной цифровой клавиатуры: нажатие клавиши даёт новую строку суммы. Знаков после запятой не больше,
// чем у валюты (units); у валюты без дробной части запятой нет.
const MAX_LEN = 14

export function pressKey(value: string, key: string, units: number): string {
  if (key === 'back') return value.slice(0, -1)
  const sepAt = value.search(/[.,]/)
  if (key === ',') {
    if (units === 0 || sepAt >= 0) return value
    return value === '' ? '0,' : `${value},`
  }
  if (!/^\d$/.test(key) || value.length >= MAX_LEN) return value
  if (sepAt >= 0 && value.length - sepAt - 1 >= units) return value
  if (value === '0') return key
  return value + key
}
