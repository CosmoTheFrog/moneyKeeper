import { describe, expect, it } from 'vitest'
import { fillPercent, monthPace, parseBudgeted, rowStatus, toBudgetTone } from './budget'

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

describe('monthPace', () => {
  it.each([
    ['2026-11', '2026-11-15', 50],
    ['2026-11', '2026-11-30', 100],
    ['2026-02', '2026-02-14', 50],
    ['2026-10', '2026-11-15', null],
    ['2026-12', '2026-11-15', null],
  ])('%s при сегодня %s → %s', (month, today, want) => {
    expect(monthPace(month, today)).toBe(want)
  })
})

describe('rowStatus', () => {
  const row = (budgeted: number, spent: number, balance: number) => ({ category_id: 1, budgeted_minor: budgeted, spent_minor: spent, balance_minor: balance })
  it.each([
    ['в норме', row(1000, 250, 750), 50, 'ok'],
    ['ничего не потрачено', row(1000, 0, 1000), 50, 'ok'],
    ['быстрее календаря', row(1000, 700, 300), 50, 'warn'],
    ['почти у лимита без темпа', row(1000, 950, 50), null, 'warn'],
    ['вровень с календарём', row(1000, 600, 400), 50, 'ok'],
    ['всё потрачено', row(1000, 1000, 0), 50, 'done'],
    ['перерасход', row(1000, 1200, -200), 50, 'over'],
    ['траты без плана', row(0, 500, -500), 50, 'noplan'],
    ['без плана, но покрыто переносом', row(0, 500, 300), 50, 'ok'],
    ['пустая строка', row(0, 0, 0), 50, 'ok'],
  ])('%s', (_name, r, pace, want) => {
    expect(rowStatus(r, pace)).toBe(want)
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
