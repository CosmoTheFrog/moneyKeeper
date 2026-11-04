import { describe, expect, it } from 'vitest'
import { compareMonths, fillPercent, parseBudgeted, spendingOf, suggestBudget, toBudgetTone } from './budget'
import type { BudgetMonth } from './api'

describe('toBudgetTone', () => {
  it.each([
    [100, 'open'],
    [1, 'open'],
    [0, 'done'],
    [-1, 'over'],
    [-50000, 'over'],
  ])('%i → %s', (minor, want) => {
    expect(toBudgetTone(minor)).toBe(want)
  })
})

describe('fillPercent', () => {
  it.each([
    [0, 1000, 0],
    [250, 1000, 25],
    [333, 1000, 33],
    [1000, 1000, 100],
    [1500, 1000, 100],
    [500, 0, 100],
    [0, 0, 0],
    [-100, 1000, 0],
  ])('spent %i of %i → %i%%', (spent, budgeted, want) => {
    expect(fillPercent(spent, budgeted)).toBe(want)
  })
})

describe('parseBudgeted', () => {
  it.each([
    ['', 0],
    ['  ', 0],
    ['0', 0],
    ['0,00', 0],
    ['0.0', 0],
    ['500', 50000],
    ['12,5', 1250],
  ])('%s → %i', (input, want) => {
    expect(parseBudgeted(input, 'GEL')).toBe(want)
  })

  it.each(['abc', '-5', '1,234', '5 EUR'])('%s → null', (input) => {
    expect(parseBudgeted(input, 'GEL')).toBeNull()
  })
})

const month = (rows: [number, number, number][]): BudgetMonth => ({
  month: '2026-10',
  currency: 'GEL',
  income_minor: 0,
  budgeted_minor: 0,
  to_budget_minor: 0,
  missing_rates: 0,
  categories: rows.map(([id, budgeted, spent]) => ({ category_id: id, budgeted_minor: budgeted, spent_minor: spent, balance_minor: budgeted - spent })),
})

describe('compareMonths', () => {
  it('сортирует по изменению траты и пропускает пустые категории', () => {
    const now = month([[1, 1000, 500], [2, 0, 0], [3, 0, 900], [4, 200, 200]])
    const then = month([[1, 1000, 800], [2, 0, 0], [3, 0, 100], [4, 0, 0]])
    expect(compareMonths(now, then).map((r) => [r.categoryId, r.spentDelta])).toEqual([
      [3, 800],
      [1, -300],
      [4, 200],
    ])
  })

  it('категория, которой не было в прошлом месяце, сравнивается с нулём', () => {
    const rows = compareMonths(month([[7, 100, 50]]), month([]))
    expect(rows).toEqual([{ categoryId: 7, budgetedNow: 100, spentNow: 50, budgetedThen: 0, spentThen: 0, spentDelta: 50 }])
  })
})

describe('suggestBudget', () => {
  it('усредняет по месяцам с тратами и округляет', () => {
    const got = suggestBudget([month([[1, 0, 1000], [2, 0, 100]]), month([[1, 0, 500]]), month([[1, 0, 0]])])
    expect(got).toEqual([
      { categoryId: 1, minor: 750 },
      { categoryId: 2, minor: 50 },
    ])
  })

  it('без трат в истории предложений нет', () => {
    expect(suggestBudget([month([[1, 500, 0]]), month([])])).toEqual([])
  })
})

describe('spendingOf', () => {
  it('сортирует по трате, считает итог и сравнивает с прошлым месяцем', () => {
    const now = month([[1, 0, 300], [2, 500, 900], [3, 0, 0]])
    const then = month([[1, 0, 100], [2, 0, 200], [4, 0, 50]])
    expect(spendingOf(now, then)).toEqual({
      rows: [
        { categoryId: 2, spentMinor: 900, previousMinor: 200 },
        { categoryId: 1, spentMinor: 300, previousMinor: 100 },
      ],
      totalMinor: 1200,
      previousTotalMinor: 350,
    })
  })

  it('без прошлого месяца сравнивать не с чем', () => {
    const got = spendingOf(month([[1, 0, 300]]), null)
    expect(got.previousTotalMinor).toBe(0)
    expect(got.rows).toEqual([{ categoryId: 1, spentMinor: 300, previousMinor: 0 }])
  })

  it('заложенная сумма без трат и возврат средств в список не попадают', () => {
    expect(spendingOf(month([[1, 5000, 0], [2, 0, -200]]), null).rows).toEqual([])
  })
})
