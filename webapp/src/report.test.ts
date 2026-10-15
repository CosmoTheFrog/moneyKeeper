import { describe, expect, it } from 'vitest'
import type { Category, CategoryGroup } from './api'
import type { Entry } from './ledger'
import {
  CIRCLE,
  addDays,
  allTimeRange,
  arcs,
  diffDays,
  inGroup,
  missingRates,
  percent,
  rangeFor,
  resolveRange,
  spendBuckets,
  spendByCategory,
  spendByGroup,
  spendOf,
  spendTotals,
  type Measure,
} from './report'

const CRITICAL = 1
const DAILY = 3
const GOAL = 5

const groups: CategoryGroup[] = [
  { id: CRITICAL, name: 'Критичные' },
  { id: 2, name: 'Долги' },
  { id: DAILY, name: 'Ежедневные' },
  { id: GOAL, name: 'Цели' },
  { id: 6, name: 'Своя' },
]

function cat(id: number, name: string, groupId: number): Category {
  return { id, name, group_id: groupId, description: '', hidden: false, rollover: false }
}

// В GEL базовая сумма равна самой сумме, для остальных валют курса по умолчанию нет.
function exp(id: number, categoryId: number, amountMinor: number, date: string, currency = 'GEL', baseMinor: number | null = currency === 'GEL' ? amountMinor : null): Entry {
  return { id, kind: 'expense', date, amountMinor, currency, baseMinor, accountId: 1, toAccountId: null, categoryId, note: '' }
}

const GEL: Measure = { kind: 'currency', code: 'GEL' }
const EUR: Measure = { kind: 'currency', code: 'EUR' }
const BASE: Measure = { kind: 'base' }

const categories = [cat(1, 'Аренда', CRITICAL), cat(2, 'Продукты', DAILY), cat(3, 'Кафе', DAILY)]

describe('addDays', () => {
  it.each([
    ['2026-10-12', -6, '2026-10-06'],
    ['2026-10-03', -6, '2026-09-27'],
    ['2026-01-02', -6, '2025-12-27'],
    ['2024-03-01', -1, '2024-02-29'],
    ['2026-10-31', 1, '2026-11-01'],
    ['2026-10-12', 0, '2026-10-12'],
  ])('%s %i → %s', (day, n, want) => {
    expect(addDays(day, n)).toBe(want)
  })
})

describe('rangeFor', () => {
  it('counts back from today', () => {
    expect(rangeFor('week', '2026-10-12')).toEqual({ from: '2026-10-06', to: '2026-10-12' })
    expect(rangeFor('twoWeeks', '2026-10-12')).toEqual({ from: '2026-09-29', to: '2026-10-12' })
    expect(rangeFor('threeWeeks', '2026-10-12')).toEqual({ from: '2026-09-22', to: '2026-10-12' })
  })

  it('takes the calendar month for month', () => {
    expect(rangeFor('month', '2026-10-12')).toEqual({ from: '2026-10-01', to: '2026-10-12' })
  })
})

describe('spendByCategory', () => {
  const range = { from: '2026-10-01', to: '2026-10-31' }

  it('sums expenses per category, biggest first', () => {
    const entries = [exp(1, 2, -500, '2026-10-02'), exp(2, 2, -300, '2026-10-05'), exp(3, 1, -90000, '2026-10-01'), exp(4, 3, -800, '2026-10-09')]
    const out = spendByCategory(entries, categories, range, GEL)
    expect(out.map((s) => [s.label, s.totalMinor])).toEqual([
      ['Аренда', 90000],
      ['Кафе', 800],
      ['Продукты', 800],
    ])
  })

  it('ignores other currencies, other kinds and dates outside the range', () => {
    const entries: Entry[] = [
      exp(1, 2, -500, '2026-10-02'),
      exp(2, 2, -700, '2026-10-02', 'EUR'),
      exp(3, 2, -900, '2026-09-30'),
      exp(4, 2, -900, '2026-11-01'),
      { ...exp(5, 2, 100, '2026-10-03'), kind: 'income', categoryId: null },
    ]
    expect(spendByCategory(entries, categories, range, GEL).map((s) => s.totalMinor)).toEqual([500])
    expect(spendByCategory(entries, categories, range, EUR).map((s) => s.totalMinor)).toEqual([700])
  })

  it('includes both ends of the range', () => {
    const entries = [exp(1, 2, -100, '2026-10-01'), exp(2, 2, -200, '2026-10-31')]
    expect(spendByCategory(entries, categories, range, GEL)[0]?.totalMinor).toBe(300)
  })

  it('returns nothing when there are no expenses', () => {
    expect(spendByCategory([], categories, range, GEL)).toEqual([])
  })
})

describe('spendByGroup', () => {
  it('sums categories into groups, biggest first', () => {
    const slices = spendByCategory(
      [exp(1, 2, -500, '2026-10-02'), exp(2, 3, -800, '2026-10-03'), exp(3, 1, -90000, '2026-10-01')],
      categories,
      { from: '2026-10-01', to: '2026-10-31' },
      GEL,
    )
    expect(spendByGroup(slices, groups).map((g) => [g.label, g.totalMinor])).toEqual([
      ['Критичные', 90000],
      ['Ежедневные', 1300],
    ])
  })

  it('knows groups created by the user', () => {
    const own = [...categories, cat(7, 'Авиабилеты', 6)]
    const slices = spendByCategory([exp(1, 7, -4000, '2026-10-02')], own, { from: '2026-10-01', to: '2026-10-31' }, GEL)
    expect(spendByGroup(slices, groups).map((g) => [g.label, g.totalMinor])).toEqual([['Своя', 4000]])
  })
})

describe('spendTotals', () => {
  const range = { from: '2026-10-01', to: '2026-10-31' }

  it('sums expenses per currency, base currency first, then by amount', () => {
    const entries = [
      exp(1, 2, -500, '2026-10-02'),
      exp(2, 2, -2500, '2026-10-03', 'USD'),
      exp(3, 2, -700, '2026-10-04', 'EUR'),
      exp(4, 2, -300, '2026-10-05'),
    ]
    expect(spendTotals(entries, range, 'GEL')).toEqual([
      { currency: 'GEL', totalMinor: 800 },
      { currency: 'USD', totalMinor: 2500 },
      { currency: 'EUR', totalMinor: 700 },
    ])
  })

  it('does not put the base currency first when it has no spending', () => {
    expect(spendTotals([exp(1, 2, -700, '2026-10-04', 'EUR')], range, 'GEL')).toEqual([{ currency: 'EUR', totalMinor: 700 }])
  })

  it('ignores income, transfers, openings and dates outside the range', () => {
    const entries: Entry[] = [
      { ...exp(1, 2, 900, '2026-10-02'), kind: 'income', categoryId: null },
      { ...exp(2, 2, 900, '2026-10-02'), kind: 'opening', categoryId: null },
      { ...exp(3, 2, 900, '2026-10-02'), kind: 'transfer', categoryId: null },
      exp(4, 2, -100, '2026-09-30'),
      exp(5, 2, -100, '2026-11-01'),
    ]
    expect(spendTotals(entries, range, 'GEL')).toEqual([])
  })
})

describe('inGroup', () => {
  it('keeps only the categories of one group, biggest first', () => {
    const slices = spendByCategory(
      [exp(1, 2, -500, '2026-10-02'), exp(2, 3, -800, '2026-10-03'), exp(3, 1, -90000, '2026-10-01')],
      categories,
      { from: '2026-10-01', to: '2026-10-31' },
      GEL,
    )
    expect(inGroup(slices, DAILY).map((s) => [s.label, s.totalMinor])).toEqual([
      ['Кафе', 800],
      ['Продукты', 500],
    ])
    expect(inGroup(slices, CRITICAL).map((s) => s.label)).toEqual(['Аренда'])
    expect(inGroup(slices, GOAL)).toEqual([])
  })
})

describe('spending in the base currency', () => {
  const range = { from: '2026-10-01', to: '2026-10-31' }
  // 10 € = 31,59 ₾ по курсу дня, 5 $ = 13,54 ₾, у третьей записи курса нет.
  const entries = [exp(1, 2, -1000, '2026-10-02', 'EUR', -3159), exp(2, 3, -500, '2026-10-03', 'USD', -1354), exp(3, 2, -700, '2026-10-04', 'USDT'), exp(4, 2, -400, '2026-10-05')]

  it('spendOf takes the base amount or the own amount of one currency', () => {
    expect(spendOf(entries[0]!, BASE)).toBe(3159)
    expect(spendOf(entries[2]!, BASE)).toBeNull()
    expect(spendOf(entries[0]!, EUR)).toBe(1000)
    expect(spendOf(entries[0]!, GEL)).toBeNull()
    expect(spendOf({ ...entries[0]!, kind: 'income' }, BASE)).toBeNull()
  })

  it('sums all currencies by their own day rate', () => {
    const out = spendByCategory(entries, categories, range, BASE)
    expect(out.map((s) => [s.label, s.totalMinor])).toEqual([
      ['Продукты', 3159 + 400],
      ['Кафе', 1354],
    ])
  })

  it('counts spending without a rate separately', () => {
    expect(missingRates(entries, range)).toBe(1)
    expect(missingRates(entries, { from: '2026-10-05', to: '2026-10-31' })).toBe(0)
  })
})

describe('allTimeRange', () => {
  it('starts at the first expense', () => {
    const entries = [exp(1, 2, -100, '2026-08-17'), exp(2, 2, -100, '2026-10-01'), { ...exp(3, 2, 100, '2026-01-01'), kind: 'income' as const }]
    expect(allTimeRange(entries, '2026-10-12')).toEqual({ from: '2026-08-17', to: '2026-10-12' })
  })

  it('is a single day without expenses', () => {
    expect(allTimeRange([], '2026-10-12')).toEqual({ from: '2026-10-12', to: '2026-10-12' })
  })
})

describe('resolveRange', () => {
  const custom = { from: '2026-09-15', to: '2026-10-03' }
  const entries = [exp(1, 2, -100, '2026-08-17')]

  it('takes presets, the calendar range and all time', () => {
    expect(resolveRange('week', custom, '2026-10-12', entries)).toEqual({ from: '2026-10-06', to: '2026-10-12' })
    expect(resolveRange('custom', custom, '2026-10-12', entries)).toEqual(custom)
    expect(resolveRange('all', custom, '2026-10-12', entries)).toEqual({ from: '2026-08-17', to: '2026-10-12' })
  })
})

describe('diffDays', () => {
  it('counts calendar days across months and leap years', () => {
    expect(diffDays('2026-10-06', '2026-10-12')).toBe(6)
    expect(diffDays('2026-09-30', '2026-10-01')).toBe(1)
    expect(diffDays('2024-02-28', '2024-03-01')).toBe(2)
    expect(diffDays('2026-10-12', '2026-10-12')).toBe(0)
  })
})

describe('spendBuckets', () => {
  it('splits a short period into days, keeping empty ones', () => {
    const range = { from: '2026-10-01', to: '2026-10-05' }
    const out = spendBuckets([exp(1, 2, -500, '2026-10-02'), exp(2, 2, -300, '2026-10-02'), exp(3, 2, -100, '2026-10-05')], range, GEL)
    expect(out.unit).toBe('day')
    expect(out.items.map((b) => b.totalMinor)).toEqual([0, 800, 0, 0, 100])
    expect(out.items[0]).toMatchObject({ key: '2026-10-01', from: '2026-10-01', to: '2026-10-01' })
  })

  it('goes by months for a long period and clips the ends to the period', () => {
    const range = { from: '2026-08-17', to: '2026-10-12' }
    const out = spendBuckets([exp(1, 2, -500, '2026-08-20'), exp(2, 2, -200, '2026-09-30'), exp(3, 2, -100, '2026-10-12')], range, GEL)
    expect(out.unit).toBe('month')
    expect(out.items.map((b) => [b.key, b.from, b.to, b.totalMinor])).toEqual([
      ['2026-08', '2026-08-17', '2026-08-31', 500],
      ['2026-09', '2026-09-01', '2026-09-30', 200],
      ['2026-10', '2026-10-01', '2026-10-12', 100],
    ])
  })

  it('switches to months after 31 days', () => {
    expect(spendBuckets([], { from: '2026-10-01', to: '2026-10-31' }, GEL).unit).toBe('day')
    expect(spendBuckets([], { from: '2026-10-01', to: '2026-11-01' }, GEL).unit).toBe('month')
  })

  it('crosses the year boundary', () => {
    const out = spendBuckets([], { from: '2025-11-15', to: '2026-02-10' }, GEL)
    expect(out.items.map((b) => b.key)).toEqual(['2025-11', '2025-12', '2026-01', '2026-02'])
  })

  it('ignores spending outside the period and other currencies', () => {
    const range = { from: '2026-10-01', to: '2026-10-03' }
    const out = spendBuckets([exp(1, 2, -500, '2026-09-30'), exp(2, 2, -500, '2026-10-02', 'EUR'), exp(3, 2, -100, '2026-10-02')], range, GEL)
    expect(out.items.map((b) => b.totalMinor)).toEqual([0, 100, 0])
  })
})

describe('percent and arcs', () => {
  it('rounds shares', () => {
    expect(percent(1, 3)).toBe(33)
    expect(percent(2, 3)).toBe(67)
    expect(percent(0, 0)).toBe(0)
  })

  it('splits the circle proportionally', () => {
    const a = arcs([50, 30, 20])
    expect(a).toHaveLength(3)
    expect(a[0]?.offset).toBe(-0)
    expect(a[0]?.length).toBeCloseTo(CIRCLE * 0.5)
    expect(a[1]?.offset).toBeCloseTo(-CIRCLE * 0.5)
    expect(a[2]?.offset).toBeCloseTo(-CIRCLE * 0.8)
    expect(a.reduce((s, x) => s + x.length, 0)).toBeCloseTo(CIRCLE)
  })

  it('returns nothing for zero total', () => {
    expect(arcs([])).toEqual([])
    expect(arcs([0, 0])).toEqual([])
  })
})
