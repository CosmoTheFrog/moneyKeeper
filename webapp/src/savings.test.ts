import { describe, expect, it } from 'vitest'
import { monthsUntil, perMonth, progressPercent } from './savings'

describe('progressPercent', () => {
  it.each([
    [120000, 250000, 48],
    [0, 250000, 0],
    [250000, 250000, 100],
    [300000, 250000, 100],
    [-500, 250000, 0],
    [100, null, 0],
    [100, 0, 0],
    [249999, 250000, 99],
  ])('saved %i of %s → %i%%', (saved, target, want) => {
    expect(progressPercent(saved, target)).toBe(want)
  })
})

describe('monthsUntil', () => {
  it.each([
    ['2026-10-12', '2027-05-01', 7],
    ['2026-10-12', '2026-11-01', 1],
    ['2026-10-12', '2026-12-31', 2],
    ['2026-10-12', '2027-10-12', 12],
    ['2026-10-12', '2026-10-12', 1],
    ['2026-10-12', '2026-09-01', 1],
    ['2026-12-20', '2027-01-05', 1],
  ])('%s → %s = %i', (today, target, want) => {
    expect(monthsUntil(today, target)).toBe(want)
  })
})

describe('perMonth', () => {
  it.each([
    [130000, 5, 26000],
    [100, 3, 34],
    [0, 4, 0],
    [-100, 4, 0],
    [100, 0, 100],
  ])('%i over %i months → %i', (remaining, months, want) => {
    expect(perMonth(remaining, months)).toBe(want)
  })
})
