import type { Category, CategoryGroup } from './api'
import type { Entry } from './ledger'

// Диапазон дат включительно, обе границы в виде YYYY-MM-DD. Строки этого вида сравниваются как даты.
export type Range = { from: string; to: string }

export type PresetPeriod = 'week' | 'twoWeeks' | 'threeWeeks' | 'month'

export const groupOrder: CategoryGroup[] = ['critical', 'debt', 'daily', 'rainy', 'goal']

export function addDays(day: string, n: number): string {
  const [y, m, d] = day.split('-').map(Number)
  return new Date(Date.UTC(y ?? 1970, (m ?? 1) - 1, (d ?? 1) + n)).toISOString().slice(0, 10)
}

// Неделя, две и три недели считаются назад от сегодня, месяц это текущий календарный месяц.
export function rangeFor(period: PresetPeriod, today: string): Range {
  switch (period) {
    case 'week':
      return { from: addDays(today, -6), to: today }
    case 'twoWeeks':
      return { from: addDays(today, -13), to: today }
    case 'threeWeeks':
      return { from: addDays(today, -20), to: today }
    case 'month':
      return { from: `${today.slice(0, 8)}01`, to: today }
  }
}

export type Slice = {
  key: string
  label: string
  totalMinor: number
  group: CategoryGroup
}

// spendByCategory суммирует расходы за период в одной валюте: курсов пока нет, разные валюты не складываем.
export function spendByCategory(entries: Entry[], categories: Category[], range: Range, currency: string): Slice[] {
  const byId = new Map(categories.map((c) => [c.id, c]))
  const totals = new Map<number, number>()
  for (const e of entries) {
    if (e.kind !== 'expense' || e.categoryId === null || e.currency !== currency) continue
    if (e.date < range.from || e.date > range.to) continue
    totals.set(e.categoryId, (totals.get(e.categoryId) ?? 0) - e.amountMinor)
  }

  const out: Slice[] = []
  for (const [id, totalMinor] of totals) {
    const c = byId.get(id)
    if (c !== undefined) out.push({ key: String(id), label: c.name, totalMinor, group: c.group })
  }
  return out.sort((a, b) => b.totalMinor - a.totalMinor || a.label.localeCompare(b.label, 'ru'))
}

export function spendByGroup(slices: Slice[], label: (g: CategoryGroup) => string): Slice[] {
  const totals = new Map<CategoryGroup, number>()
  for (const s of slices) totals.set(s.group, (totals.get(s.group) ?? 0) + s.totalMinor)
  return groupOrder
    .filter((g) => totals.has(g))
    .map((g) => ({ key: g, label: label(g), totalMinor: totals.get(g) ?? 0, group: g }))
    .sort((a, b) => b.totalMinor - a.totalMinor)
}

export function percent(part: number, total: number): number {
  return total === 0 ? 0 : Math.round((part * 100) / total)
}

export const RADIUS = 70
export const CIRCLE = 2 * Math.PI * RADIUS

export type Arc = { length: number; offset: number }

// arcs считает дуги кольца: длина пропорциональна доле, смещение накапливается со знаком минус.
// Это только для рисования, суммы остаются целыми.
export function arcs(values: number[]): Arc[] {
  const total = values.reduce((a, b) => a + b, 0)
  if (total === 0) return []
  let used = 0
  return values.map((v) => {
    const length = (v / total) * CIRCLE
    const arc = { length, offset: -used }
    used += length
    return arc
  })
}
