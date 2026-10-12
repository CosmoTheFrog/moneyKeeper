import { describe, expect, it } from 'vitest'
import type { Account } from './api'
import { allows, balanceIn, pickAccount, usableCurrencies } from './accounts'

function acc(id: number, currency: string, over: Partial<Account> = {}): Account {
  return { id, name: `a${id}`, kind: 'debit', currency, multi_currency: false, is_default: false, balances: [{ currency, balance_minor: 0, spent_minor: 0, base_balance_minor: null, base_spent_minor: null }], ...over }
}

const all = ['GEL', 'EUR', 'USD', 'RUB', 'USDT']

describe('allows', () => {
  it('a plain account takes only its own currency, a multi-currency one takes any', () => {
    expect(allows(acc(1, 'GEL'), 'GEL')).toBe(true)
    expect(allows(acc(1, 'GEL'), 'EUR')).toBe(false)
    expect(allows(acc(2, 'USDT', { multi_currency: true }), 'EUR')).toBe(true)
  })
})

describe('balanceIn', () => {
  it('returns the balance for the currency, zero when there are no records', () => {
    const a = acc(1, 'USDT', {
      multi_currency: true,
      balances: [
        { currency: 'USDT', balance_minor: 500, spent_minor: 0, base_balance_minor: null, base_spent_minor: null },
        { currency: 'EUR', balance_minor: 250, spent_minor: 10, base_balance_minor: null, base_spent_minor: null },
      ],
    })
    expect(balanceIn(a, 'EUR')).toBe(250)
    expect(balanceIn(a, 'USD')).toBe(0)
  })
})

describe('usableCurrencies', () => {
  it('lists only the currencies of plain accounts', () => {
    expect(usableCurrencies([acc(1, 'GEL'), acc(2, 'EUR')], all)).toEqual(['GEL', 'EUR'])
  })

  it('opens every currency when there is a multi-currency account', () => {
    expect(usableCurrencies([acc(1, 'GEL'), acc(2, 'USDT', { multi_currency: true })], all)).toEqual(all)
  })

  it('is empty without accounts', () => {
    expect(usableCurrencies([], all)).toEqual([])
  })
})

describe('pickAccount', () => {
  const accounts = [acc(1, 'GEL', { is_default: true }), acc(2, 'EUR'), acc(3, 'USDT', { multi_currency: true }), acc(4, 'EUR')]

  it('keeps the current account when it allows the currency', () => {
    expect(pickAccount(accounts, 'EUR', 3)).toBe(3)
    expect(pickAccount(accounts, 'GEL', 1)).toBe(1)
  })

  it('moves to the default account of the currency, else the first fitting', () => {
    expect(pickAccount(accounts, 'GEL', 2)).toBe(1) // основной счёт GEL
    expect(pickAccount(accounts, 'EUR', 1)).toBe(2) // основного в EUR нет, первый подходящий
  })

  it('returns 0 when nothing fits', () => {
    expect(pickAccount([acc(1, 'GEL')], 'EUR', 1)).toBe(0)
  })
})
