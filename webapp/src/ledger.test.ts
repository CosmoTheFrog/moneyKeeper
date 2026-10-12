import { describe, expect, it } from 'vitest'
import type { Transaction } from './api'
import { dayIn, groupByDay, toEntries } from './ledger'

function tx(over: Partial<Transaction> & Pick<Transaction, 'id'>): Transaction {
  return {
    account_id: 1,
    category_id: null,
    kind: 'income',
    amount_minor: 100,
    currency: 'GEL',
    base_minor: null,
    date: '2026-10-12',
    note: '',
    transfer_id: null,
    ...over,
  }
}

describe('toEntries', () => {
  it('merges both legs of a transfer into one entry', () => {
    const out = toEntries([
      tx({ id: 4, kind: 'transfer', account_id: 2, amount_minor: 3000, transfer_id: 9 }),
      tx({ id: 3, kind: 'transfer', account_id: 1, amount_minor: -3000, transfer_id: 9 }),
      tx({ id: 2, kind: 'expense', category_id: 5, amount_minor: -500 }),
    ])
    expect(out).toHaveLength(2)
    expect(out[0]).toMatchObject({ id: 3, kind: 'transfer', accountId: 1, toAccountId: 2, amountMinor: 3000 })
    expect(out[1]).toMatchObject({ id: 2, kind: 'expense', amountMinor: -500, categoryId: 5 })
  })

  it('keeps a lone leg when the other one is outside the page', () => {
    const out = toEntries([tx({ id: 4, kind: 'transfer', account_id: 2, amount_minor: 3000, transfer_id: 9 })])
    expect(out).toHaveLength(1)
    expect(out[0]).toMatchObject({ id: 4, accountId: 2, toAccountId: null, amountMinor: 3000 })
  })

  it('does not mix legs of different transfers', () => {
    const out = toEntries([
      tx({ id: 1, kind: 'transfer', account_id: 1, amount_minor: -100, transfer_id: 1 }),
      tx({ id: 2, kind: 'transfer', account_id: 2, amount_minor: 100, transfer_id: 1 }),
      tx({ id: 3, kind: 'transfer', account_id: 1, amount_minor: -200, transfer_id: 2 }),
      tx({ id: 4, kind: 'transfer', account_id: 3, amount_minor: 200, transfer_id: 2 }),
    ])
    expect(out.map((e) => [e.id, e.toAccountId])).toEqual([
      [1, 2],
      [3, 3],
    ])
  })
})

describe('toEntries base amounts', () => {
  it('carries the base amount, absolute for transfers', () => {
    const out = toEntries([
      tx({ id: 2, kind: 'expense', category_id: 5, amount_minor: -1000, currency: 'EUR', base_minor: -3159 }),
      tx({ id: 3, kind: 'transfer', account_id: 1, amount_minor: -1000, currency: 'EUR', base_minor: -3159, transfer_id: 9 }),
      tx({ id: 4, kind: 'transfer', account_id: 2, amount_minor: 1000, currency: 'EUR', base_minor: 3159, transfer_id: 9 }),
      tx({ id: 1, kind: 'expense', category_id: 5, amount_minor: -700, currency: 'USDT' }),
    ])
    expect(out.map((e) => [e.id, e.baseMinor])).toEqual([
      [2, -3159],
      [3, 3159],
      [1, null],
    ])
  })
})

describe('dayIn', () => {
  // 21:30 UTC 11 октября: в Тбилиси (UTC+4) уже 12-е, в Нью-Йорке (UTC-4) ещё 11-е.
  const now = new Date('2026-10-11T21:30:00Z')

  it.each([
    ['UTC', '2026-10-11'],
    ['Asia/Tbilisi', '2026-10-12'],
    ['America/New_York', '2026-10-11'],
  ])('%s → %s', (zone, want) => {
    expect(dayIn(zone, now)).toBe(want)
  })
})

describe('groupByDay', () => {
  it('groups neighbours with the same date and keeps order', () => {
    const entries = toEntries([
      tx({ id: 3, date: '2026-10-12' }),
      tx({ id: 2, date: '2026-10-12' }),
      tx({ id: 1, date: '2026-10-01' }),
    ])
    const groups = groupByDay(entries)
    expect(groups.map((g) => [g.date, g.entries.map((e) => e.id)])).toEqual([
      ['2026-10-12', [3, 2]],
      ['2026-10-01', [1]],
    ])
  })

  it('returns nothing for an empty list', () => {
    expect(groupByDay([])).toEqual([])
  })
})
