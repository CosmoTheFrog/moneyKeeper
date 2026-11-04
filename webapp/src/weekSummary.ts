import type { Category } from './api'
import type { Entry } from './ledger'
import { rangeFor, spendByCategory, spendOf, type Measure, type Slice } from './report'

export type WeekSummary = {
  // Траты последних семи дней и семи дней перед ними в базовой валюте по курсу дня записи.
  totalMinor: number
  previousMinor: number
  // Категория с самой большой тратой недели, null если трат нет.
  top: Slice | null
  // Траты недели без курса: в сумму не вошли.
  missingRates: number
}

const BASE: Measure = { kind: 'base' }

// weekSummary сводит неделю, заканчивающуюся сегодня, и сравнивает её с предыдущей. null, если за обе недели расходов нет.
export function weekSummary(entries: Entry[], categories: Category[], today: string): WeekSummary | null {
  const week = rangeFor('week', today)
  const before = rangeFor('week', today, -1)
  const sum = (from: string, to: string) =>
    entries.reduce((s, e) => (e.date >= from && e.date <= to ? s + (spendOf(e, BASE) ?? 0) : s), 0)
  const expenses = (from: string, to: string) => entries.filter((e) => e.kind === 'expense' && e.date >= from && e.date <= to)

  const thisWeek = expenses(week.from, week.to)
  if (thisWeek.length === 0 && expenses(before.from, before.to).length === 0) return null
  return {
    totalMinor: sum(week.from, week.to),
    previousMinor: sum(before.from, before.to),
    top: spendByCategory(entries, categories, week, BASE)[0] ?? null,
    missingRates: thisWeek.filter((e) => e.baseMinor === null).length,
  }
}
