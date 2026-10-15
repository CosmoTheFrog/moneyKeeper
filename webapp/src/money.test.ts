import { beforeAll, describe, expect, it } from 'vitest'
import { dealPrice, formatInput, formatMinor, formatRate, isCoin, isStock, parseAmount, registerCurrencies, unitsOf } from './money'

beforeAll(() => {
  registerCurrencies([
    { code: 'GEL', units: 2, coin: false, stock: false },
    { code: 'BTC', units: 8, coin: true, stock: false },
    { code: 'DOGE', units: 4, coin: true, stock: false },
    { code: 'SHIB', units: 0, coin: true, stock: false },
    { code: 'TSLA', units: 4, coin: true, stock: true },
  ])
})

describe('formatMinor', () => {
  it.each([
    [0, 'GEL', false, '0,00 ₾'],
    [5, 'GEL', false, '0,05 ₾'],
    [123450, 'GEL', false, '1 234,50 ₾'],
    [-123450, 'EUR', false, '−1 234,50 €'],
    [100000000, 'USD', false, '1 000 000,00 $'],
    [2000, 'GEL', true, '+20,00 ₾'],
    [-2000, 'GEL', true, '−20,00 ₾'],
    [0, 'GEL', true, '0,00 ₾'],
    [150, 'USDT', false, '1,50 USDT'],
    [150, 'XXX', false, '1,50 XXX'],
    [100000, 'BTC', false, '0,00100000 BTC'],
    [-12345678901, 'BTC', false, '−123,45678901 BTC'],
    [25000, 'DOGE', true, '+2,5000 DOGE'],
    [1500000, 'SHIB', false, '1 500 000 SHIB'],
    [0, 'SHIB', false, '0 SHIB'],
  ])('%i %s signed=%s → %s', (minor, currency, signed, want) => {
    expect(formatMinor(minor, currency, signed)).toBe(want)
  })
})

describe('parseAmount', () => {
  it.each([
    ['12', 'GEL', 1200],
    ['12,5', 'GEL', 1250],
    ['12.50', 'GEL', 1250],
    [' 1 200,05 ', 'GEL', 120005],
    ['0,01', 'GEL', 1],
    ['10000000000', 'GEL', 1_000_000_000_000],
    ['0,00125', 'BTC', 125000],
    ['0.00000001', 'BTC', 1],
    ['10000', 'BTC', 1_000_000_000_000],
    ['2,5', 'DOGE', 25000],
    ['1500000', 'SHIB', 1_500_000],
  ])('%s %s → %i', (input, currency, want) => {
    expect(parseAmount(input, currency)).toBe(want)
  })

  it.each([
    ['', 'GEL'],
    ['0', 'GEL'],
    ['0,00', 'GEL'],
    ['-5', 'GEL'],
    ['5,555', 'GEL'],
    ['abc', 'GEL'],
    ['5 EUR', 'GEL'],
    ['10000000001', 'GEL'],
    ['1,2,3', 'GEL'],
    ['0,000000001', 'BTC'],
    ['10000,00000001', 'BTC'],
    ['1,5', 'SHIB'],
    ['0', 'SHIB'],
    ['1,23456', 'DOGE'],
  ])('%s %s → null', (input, currency) => {
    expect(parseAmount(input, currency)).toBeNull()
  })
})

describe('formatInput', () => {
  it('returns the form used in the amount field', () => {
    expect(formatInput(123450, 'GEL')).toBe('1234,50')
    expect(formatInput(-5, 'GEL')).toBe('0,05')
    expect(formatInput(125000, 'BTC')).toBe('0,00125000')
    expect(formatInput(1500000, 'SHIB')).toBe('1500000')
  })

  it('round-trips through parseAmount', () => {
    for (const [minor, currency] of [[123450, 'GEL'], [125000, 'BTC'], [25000, 'DOGE'], [7, 'SHIB']] as const) {
      expect(parseAmount(formatInput(minor, currency), currency)).toBe(minor)
    }
  })
})

describe('currency registry', () => {
  it('knows precision and coins, two decimals by default', () => {
    expect(unitsOf('BTC')).toBe(8)
    expect(unitsOf('SHIB')).toBe(0)
    expect(unitsOf('XXX')).toBe(2)
    expect(isCoin('BTC')).toBe(true)
    expect(isCoin('GEL')).toBe(false)
    expect(isStock('TSLA')).toBe(true)
    expect(isStock('BTC')).toBe(false)
  })
})

describe('formatRate', () => {
  it.each([
    ['2.6005', '2,6005'],
    ['1', '1'],
    ['0.369221680697', '0,3692'],
    ['92305.420174272633', '92 305,42'],
    ['215291.5', '215 291,5'],
    ['0.00000579', '0,00000579'],
    ['0.000005794999', '0,000005795'],
    ['999.999', '1 000'],
    ['99.99999', '100'],
    ['0', '0'],
  ])('%s → %s', (rate, want) => {
    expect(formatRate(rate)).toBe(want)
  })
})
describe('dealPrice', () => {
  it.each([
    // Отдали 9 500 ₽, получили 100 $: 95 ₽ за доллар.
    [950000, 'GEL', 10000, 'GEL', '95.000000000000'],
    // Отдали 650 ₾, получили 4,0000 DOGE: 162,50 ₾ за целую монету.
    [65000, 'GEL', 40000, 'DOGE', '162.500000000000'],
    // Монета отдаётся: за 0,001 BTC получили 250 ₾, цена одного лари 0,000004 BTC.
    [100000, 'BTC', 25000, 'GEL', '0.000004000000'],
    [0, 'GEL', 100, 'GEL', ''],
    [100, 'GEL', 0, 'GEL', ''],
  ])('%i %s for %i %s → %s', (out, outCur, inn, inCur, want) => {
    expect(dealPrice(out, outCur, inn, inCur)).toBe(want)
  })

  it('shows the price of one whole coin in what was given', () => {
    // Отдали 250,00 GEL, получили 0,00100000 BTC: 250 000 GEL за 1 BTC.
    expect(formatRate(dealPrice(25000, 'GEL', 100000, 'BTC'))).toBe('250 000')
    // Отдали 0,00100000 BTC, получили 250,00 GEL: за один лари отдали 0,000004 BTC.
    expect(formatRate(dealPrice(100000, 'BTC', 25000, 'GEL'))).toBe('0,000004')
  })
})