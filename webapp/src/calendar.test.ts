import { describe, expect, it } from 'vitest'
import { daysInRange, formatRange, monthGrid, monthTitle, pickDay, shiftMonth } from './calendar'
import { plural, t } from './i18n'

describe('pickDay', () => {
  it('first tap starts a period, second closes it', () => {
    const started = pickDay({ from: '2026-10-01', to: '2026-10-12' }, '2026-10-05')
    expect(started).toEqual({ from: '2026-10-05', to: null })
    expect(pickDay(started, '2026-10-09')).toEqual({ from: '2026-10-05', to: '2026-10-09' })
  })

  it('swaps the ends when the second tap is earlier', () => {
    expect(pickDay({ from: '2026-10-09', to: null }, '2026-10-02')).toEqual({ from: '2026-10-02', to: '2026-10-09' })
  })

  it('allows a single day', () => {
    expect(pickDay({ from: '2026-10-09', to: null }, '2026-10-09')).toEqual({ from: '2026-10-09', to: '2026-10-09' })
  })
})

describe('shiftMonth', () => {
  it.each([
    ['2026-10', 1, '2026-11'],
    ['2026-12', 1, '2027-01'],
    ['2026-01', -1, '2025-12'],
    ['2026-10', -13, '2025-09'],
    ['2026-10', 0, '2026-10'],
  ])('%s %i → %s', (ym, delta, want) => {
    expect(shiftMonth(ym, delta)).toBe(want)
  })
})

describe('monthGrid', () => {
  it('has six weeks starting on Monday', () => {
    // 1 октября 2026 года, четверг: сетка начинается с понедельника 28 сентября.
    const grid = monthGrid('2026-10')
    expect(grid).toHaveLength(42)
    expect(grid[0]).toEqual({ day: '2026-09-28', inMonth: false })
    expect(grid[3]).toEqual({ day: '2026-10-01', inMonth: true })
    expect(grid[33]).toEqual({ day: '2026-10-31', inMonth: true })
    expect(grid[34]).toEqual({ day: '2026-11-01', inMonth: false })
  })

  it('starts on the first day when it is Monday', () => {
    // 1 июня 2026 года, понедельник.
    expect(monthGrid('2026-06')[0]).toEqual({ day: '2026-06-01', inMonth: true })
  })

  it('counts exactly the days of the month as inMonth', () => {
    expect(monthGrid('2024-02').filter((c) => c.inMonth)).toHaveLength(29)
    expect(monthGrid('2026-02').filter((c) => c.inMonth)).toHaveLength(28)
  })
})

describe('formatRange', () => {
  const today = '2026-10-12'

  it.each([
    [{ from: '2026-10-12', to: '2026-10-12' }, '12 окт'],
    [{ from: '2026-10-06', to: '2026-10-12' }, '6 – 12 окт'],
    [{ from: '2026-09-12', to: '2026-10-12' }, '12 сен – 12 окт'],
    [{ from: '2025-12-20', to: '2026-01-05' }, '20 дек 2025 – 5 янв'],
    [{ from: '2025-03-01', to: '2025-03-15' }, '1 – 15 мар 2025'],
  ])('%j → %s', (range, want) => {
    expect(formatRange(range, today)).toBe(want)
  })
})

describe('helpers', () => {
  it('counts days including both ends', () => {
    expect(daysInRange({ from: '2026-10-06', to: '2026-10-12' })).toBe(7)
    expect(daysInRange({ from: '2026-10-12', to: '2026-10-12' })).toBe(1)
  })

  it('titles a month', () => {
    expect(monthTitle('2026-10')).toBe('Октябрь 2026')
  })

  it.each([
    [1, 'день'],
    [2, 'дня'],
    [5, 'дней'],
    [11, 'дней'],
    [21, 'день'],
    [112, 'дней'],
  ])('plural %i → %s', (n, want) => {
    expect(plural(n, t.calendar.days)).toBe(want)
  })
})
