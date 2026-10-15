import type { PortfolioItem } from './api'

// formatBP показывает прибыль в сотых долях процента: 1447 → «+14,47%», -250 → «−2,50%».
export function formatBP(bp: number): string {
  const abs = Math.abs(bp)
  const sign = bp < 0 ? '−' : bp > 0 ? '+' : ''
  return `${sign}${Math.trunc(abs / 100)},${String(abs % 100).padStart(2, '0')}%`
}

// bubbleSize: диаметр пузырька в пикселях по прибыли в процентах, от 56 до 150.
export function bubbleSize(bp: number | null): number {
  const percent = Math.min(Math.abs(bp ?? 0) / 100, 100)
  return Math.round(56 + percent * 0.94)
}

// shares: доли позиций в стоимости портфеля в процентах, суммируются в сто. Позиции без стоимости получают ноль.
export function shares(items: PortfolioItem[]): number[] {
  const total = items.reduce((sum, i) => sum + i.value_minor, 0)
  if (total <= 0) return items.map(() => 0)
  const raw = items.map((i) => (i.value_minor * 100) / total)
  const floors = raw.map(Math.floor)
  // Остаток до ста раздаём позициям с самой большой дробной частью, чтобы сумма сошлась.
  let left = 100 - floors.reduce((a, b) => a + b, 0)
  const order = raw.map((r, i) => ({ i, frac: r - Math.floor(r) })).sort((a, b) => b.frac - a.frac)
  for (const { i } of order) {
    if (left <= 0) break
    floors[i] = (floors[i] ?? 0) + 1
    left--
  }
  return floors
}
