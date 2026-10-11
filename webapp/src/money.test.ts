import { describe, expect, it } from 'vitest'
import { formatInput, formatMinor, parseAmount } from './money'

describe('formatMinor', () => {
  it.each([
    [0, 'GEL', false, '0,00 ₾'],
    [5, 'GEL', false, '0,05 ₾'],
    [123450, 'GEL', false, '1 234,50 ₾'],
    [-123450, 'EUR', false, '−1 234,50 €'],
    [100000000, 'USD', false, '1 000 000,00 $'],
    [2000, 'GEL', true, '+20,00 ₾'],
    [-2000, 'GEL', true, '−20,00 ₾'],
    [0, 'GEL', true, '0,00 ₾'],
    [150, 'USDT', false, '1,50 USDT'],
    [150, 'XXX', false, '1,50 XXX'],
  ])('%i %s signed=%s → %s', (minor, currency, signed, want) => {
    expect(formatMinor(minor, currency, signed)).toBe(want)
  })
})

describe('parseAmount', () => {
  it.each([
    ['12', 1200],
    ['12,5', 1250],
    ['12.50', 1250],
    [' 1 200,05 ', 120005],
    ['0,01', 1],
    ['10000000000', 1_000_000_000_000],
  ])('%s → %i', (input, want) => {
    expect(parseAmount(input)).toBe(want)
  })

  it.each(['', '0', '0,00', '-5', '5,555', 'abc', '5 EUR', '10000000001', '1,2,3'])('%s → null', (input) => {
    expect(parseAmount(input)).toBeNull()
  })
})

describe('formatInput', () => {
  it('returns the form used in the amount field', () => {
    expect(formatInput(123450)).toBe('1234,50')
    expect(formatInput(-5)).toBe('0,05')
  })
})
