import { useEffect, useState } from 'react'
import { fetchRates, type BudgetMonth } from './api'
import { unitsOf } from './money'

// useDisplayFactor: во сколько минорных единиц валюты показа превращается минорная единица базовой (по курсу дня).
// 1, пока показ в базовой валюте или курса ещё нет.
export function useDisplayFactor(display: string, base: string): number {
  const [factor, setFactor] = useState(1)
  useEffect(() => {
    if (display === base) {
      setFactor(1)
      return
    }
    let alive = true
    fetchRates(display)
      .then((table) => {
        const rate = Number(table.rates.find((r) => r.currency === base)?.rate)
        if (alive && Number.isFinite(rate) && rate > 0) setFactor(rate * 10 ** (unitsOf(display) - unitsOf(base)))
      })
      .catch(() => {})
    return () => {
      alive = false
    }
  }, [display, base])
  return display === base ? 1 : factor
}

// scaleMonth: суммы месяца, умноженные на factor; сам месяц не меняется.
export function scaleMonth(month: BudgetMonth | null, factor: number): BudgetMonth | null {
  if (month === null || factor === 1) return month
  const k = (v: number) => Math.round(v * factor)
  return {
    ...month,
    income_minor: k(month.income_minor),
    budgeted_minor: k(month.budgeted_minor),
    to_budget_minor: k(month.to_budget_minor),
    categories: month.categories.map((r) => ({ ...r, budgeted_minor: k(r.budgeted_minor), spent_minor: k(r.spent_minor), balance_minor: k(r.balance_minor) })),
  }
}
