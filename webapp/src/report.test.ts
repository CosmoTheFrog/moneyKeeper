import { describe, expect, it } from 'vitest'
import type { Category, CategoryGroup } from './api'
import type { Entry } from './ledger'
import { CIRCLE, addDays, arcs, inGroup, percent, rangeFor, spendByCategory, spendByGroup, spendTotals } from './report'

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
  return { id, name, group_id: groupId, description: '', hidden: false }
}

function exp(id: number, categoryId: number, amountMinor: number, date: string, currency = 'GEL'): Entry {
  return { id, kind: 'expense', date, amountMinor, currency, accountId: 1, toAccountId: null, categoryId, note: '' }
}

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
    const out = spendByCategory(entries, categories, range, 'GEL')
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
    expect(spendByCategory(entries, categories, range, 'GEL').map((s) => s.totalMinor)).toEqual([500])
    expect(spendByCategory(entries, categories, range, 'EUR').map((s) => s.totalMinor)).toEqual([700])
  })

  it('includes both ends of the range', () => {
    const entries = [exp(1, 2, -100, '2026-10-01'), exp(2, 2, -200, '2026-10-31')]
    expect(spendByCategory(entries, categories, range, 'GEL')[0]?.totalMinor).toBe(300)
  })

  it('returns nothing when there are no expenses', () => {
    expect(spendByCategory([], categories, range, 'GEL')).toEqual([])
  })
})

describe('spendByGroup', () => {
  it('sums categories into groups, biggest first', () => {
    const slices = spendByCategory(
      [exp(1, 2, -500, '2026-10-02'), exp(2, 3, -800, '2026-10-03'), exp(3, 1, -90000, '2026-10-01')],
      categories,
      { from: '2026-10-01', to: '2026-10-31' },
      'GEL',
    )
    expect(spendByGroup(slices, groups).map((g) => [g.label, g.totalMinor])).toEqual([
      ['Критичные', 90000],
      ['Ежедневные', 1300],
    ])
  })

  it('knows groups created by the user', () => {
    const own = [...categories, cat(7, 'Авиабилеты', 6)]
    const slices = spendByCategory([exp(1, 7, -4000, '2026-10-02')], own, { from: '2026-10-01', to: '2026-10-31' }, 'GEL')
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
      'GEL',
    )
    expect(inGroup(slices, DAILY).map((s) => [s.label, s.totalMinor])).toEqual([
      ['Кафе', 800],
      ['Продукты', 500],
    ])
    expect(inGroup(slices, CRITICAL).map((s) => s.label)).toEqual(['Аренда'])
    expect(inGroup(slices, GOAL)).toEqual([])
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
