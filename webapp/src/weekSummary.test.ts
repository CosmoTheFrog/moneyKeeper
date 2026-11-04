import { describe, expect, it } from 'vitest'
import type { Category } from './api'
import type { Entry } from './ledger'
import { weekSummary } from './weekSummary'

const categories: Category[] = [
  { id: 1, group_id: 3, name: 'Продукты', description: '', hidden: false, rollover: false },
  { id: 2, group_id: 3, name: 'Кафе', description: '', hidden: false, rollover: false },
]

let id = 0
const spend = (date: string, baseMinor: number | null, categoryId = 1): Entry => ({
  id: ++id, kind: 'expense', date, amountMinor: -(baseMinor ?? 100), currency: 'GEL', baseMinor: baseMinor === null ? null : -baseMinor,
  accountId: 1, toAccountId: null, categoryId, note: '', inMinor: null, inCurrency: null, feeMinor: 0,
})

describe('weekSummary', () => {
  const today = '2026-11-03'

  it('считает неделю до сегодня, прошлую неделю и главную категорию', () => {
    const got = weekSummary(
      [spend('2026-11-03', 1000), spend('2026-10-28', 500, 2), spend('2026-10-27', 300), spend('2026-10-20', 4000)],
      categories,
      today,
    )
    expect(got?.totalMinor).toBe(1500)
    expect(got?.previousMinor).toBe(300)
    expect(got?.top?.label).toBe('Продукты')
  })

  it('граница недели: семь дней, заканчивающихся сегодня', () => {
    const got = weekSummary([spend('2026-10-28', 100), spend('2026-10-27', 700)], categories, today)
    expect(got?.totalMinor).toBe(100)
    expect(got?.previousMinor).toBe(700)
  })

  it('трата без курса не входит в сумму и считается отдельно', () => {
    const got = weekSummary([spend('2026-11-01', 200), spend('2026-11-02', null)], categories, today)
    expect(got?.totalMinor).toBe(200)
    expect(got?.missingRates).toBe(1)
  })

  it('без расходов за две недели итога нет', () => {
    expect(weekSummary([spend('2026-10-01', 100)], categories, today)).toBeNull()
    expect(weekSummary([], categories, today)).toBeNull()
  })
})
