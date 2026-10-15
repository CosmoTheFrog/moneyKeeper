import { diffDays } from './report'

// progressPercent: какая часть цели накоплена, от нуля до ста. Без цели ноль.
export function progressPercent(savedMinor: number, targetMinor: number | null): number {
  if (targetMinor === null || targetMinor <= 0 || savedMinor <= 0) return 0
  return Math.min(100, Math.floor((savedMinor * 100) / targetMinor))
}

// monthsUntil: сколько месяцев осталось до даты, не меньше одного. Считается по календарным месяцам, неполный месяц идёт как целый.
export function monthsUntil(today: string, target: string): number {
  if (diffDays(today, target) <= 0) return 1
  const [ty, tm] = today.split('-').map(Number)
  const [gy, gm] = target.split('-').map(Number)
  const months = ((gy ?? 0) - (ty ?? 0)) * 12 + ((gm ?? 0) - (tm ?? 0))
  return Math.max(1, months)
}

// perMonth: сколько откладывать в месяц, чтобы набрать остаток за месяцы, округляя вверх до минорной единицы.
export function perMonth(remainingMinor: number, months: number): number {
  if (remainingMinor <= 0) return 0
  return Math.ceil(remainingMinor / Math.max(1, months))
}
