import { describe, expect, it } from 'vitest'
import { fillPercent, parseBudgeted, toBudgetTone } from './budget'

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
