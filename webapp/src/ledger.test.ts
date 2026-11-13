import { describe, expect, it } from 'vitest'
import type { Transaction } from './api'
import { categoryTotals, dayIn, dayNet, recordStreak, guessCategory, dayParts, daySummary, entryMatches, groupByDay, shiftDay, toEntries, type Entry } from './ledger'

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

describe('toEntries exchange', () => {
  it('merges both legs of an exchange and shows what came in', () => {
    const out = toEntries([
      tx({ id: 6, kind: 'exchange', account_id: 2, amount_minor: 10000, currency: 'USD', transfer_id: 5 }),
      tx({ id: 5, kind: 'exchange', account_id: 1, amount_minor: -950000, currency: 'RUB', transfer_id: 5 }),
    ])
    expect(out).toHaveLength(1)
    expect(out[0]).toMatchObject({ id: 5, kind: 'exchange', accountId: 1, toAccountId: 2, amountMinor: 950000, currency: 'RUB', inMinor: 10000, inCurrency: 'USD', feeMinor: 0 })
  })

  it('folds a wallet fee into the exchange and keeps a budget fee as an expense', () => {
    const out = toEntries([
      tx({ id: 9, kind: 'fee', account_id: 1, amount_minor: -50, currency: 'USDT', transfer_id: 7 }),
      tx({ id: 8, kind: 'exchange', account_id: 1, amount_minor: 100000, currency: 'BTC', transfer_id: 7 }),
      tx({ id: 7, kind: 'exchange', account_id: 1, amount_minor: -8000, currency: 'USDT', transfer_id: 7 }),
      tx({ id: 12, kind: 'expense', category_id: 3, amount_minor: -500, currency: 'RUB', transfer_id: 11 }),
      tx({ id: 11, kind: 'exchange', account_id: 1, amount_minor: -9500, currency: 'RUB', transfer_id: 11 }),
      tx({ id: 13, kind: 'exchange', account_id: 2, amount_minor: 100, currency: 'USD', transfer_id: 11 }),
    ])
    expect(out.map((e) => [e.id, e.kind, e.feeMinor])).toEqual([
      [7, 'exchange', 50],
      [12, 'expense', 0],
      [11, 'exchange', 0],
    ])
  })

  it('shows the received amount of a transfer with a network fee', () => {
    const out = toEntries([
      tx({ id: 2, kind: 'transfer', account_id: 2, amount_minor: 49990000, currency: 'BTC', transfer_id: 4 }),
      tx({ id: 1, kind: 'transfer', account_id: 1, amount_minor: -50000000, currency: 'BTC', transfer_id: 4 }),
    ])
    expect(out[0]).toMatchObject({ amountMinor: 50000000, inMinor: 49990000, inCurrency: 'BTC' })
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

function entry(over: Partial<Entry> & Pick<Entry, 'id'>): Entry {
  return {
    kind: 'expense',
    date: '2026-11-03',
    amountMinor: -100,
    currency: 'GEL',
    baseMinor: -100,
    accountId: 1,
    toAccountId: null,
    categoryId: null,
    note: '',
    inMinor: null,
    inCurrency: null,
    feeMinor: 0,
    ...over,
  }
}

describe('shiftDay', () => {
  it.each([
    ['2026-11-03', 1, '2026-11-04'],
    ['2026-11-30', 1, '2026-12-01'],
    ['2026-12-31', 1, '2027-01-01'],
    ['2026-03-01', -1, '2026-02-28'],
    ['2028-03-01', -1, '2028-02-29'],
    ['2026-11-03', 0, '2026-11-03'],
  ])('%s %i → %s', (date, delta, want) => {
    expect(shiftDay(date, delta)).toBe(want)
  })
})

describe('dayNet', () => {
  it.each([
    ['расходы и доходы', [entry({ id: 1, baseMinor: -3400 }), entry({ id: 2, kind: 'income', baseMinor: 120000 })], 116600],
    ['перевод не входит', [entry({ id: 1, baseMinor: -3400 }), entry({ id: 2, kind: 'transfer', baseMinor: 5000 })], -3400],
    ['только перевод', [entry({ id: 1, kind: 'transfer', baseMinor: 5000 })], null],
    ['нет курса у одной записи', [entry({ id: 1, baseMinor: -3400 }), entry({ id: 2, baseMinor: null })], null],
    ['пусто', [], null],
  ])('%s', (_name, entries, want) => {
    expect(dayNet(entries)).toBe(want)
  })
})

describe('dayParts', () => {
  it('складывает расходы по категориям, больше первыми', () => {
    const out = dayParts([entry({ id: 1, categoryId: 2, baseMinor: -999 }), entry({ id: 2, categoryId: 1, baseMinor: -3400 }), entry({ id: 3, categoryId: 2, baseMinor: -1 }), entry({ id: 4, kind: 'income', baseMinor: 9000 })])
    expect(out).toEqual([
      { categoryId: 1, totalMinor: 3400 },
      { categoryId: 2, totalMinor: 1000 },
    ])
  })
})

describe('entryMatches', () => {
  const accounts = new Map([[1, 'TBC'], [2, 'Наличные']])
  const categories = new Map([[7, 'Питание вне дома']])
  it.each([
    ['пустой запрос', '', true],
    ['по заметке без учёта регистра', 'PUB', true],
    ['по категории', 'питание', true],
    ['по счёту', 'tbc', true],
    ['по счёту получателя', 'налич', true],
    ['нет совпадений', 'такси', false],
  ])('%s', (_name, query, want) => {
    const e = entry({ id: 1, kind: 'transfer', note: 'Georgian Pub', categoryId: 7, toAccountId: 2 })
    expect(entryMatches(e, query, accounts, categories)).toBe(want)
  })
})

describe('daySummary', () => {
  it('считает траты и доход, перевод не входит, записи без курса считаются отдельно', () => {
    const out = daySummary([
      entry({ id: 1, baseMinor: -3400 }),
      entry({ id: 2, baseMinor: -999 }),
      entry({ id: 3, kind: 'income', baseMinor: 120000 }),
      entry({ id: 4, kind: 'transfer', baseMinor: 5000 }),
      entry({ id: 5, baseMinor: null }),
    ])
    expect(out).toEqual({ spentMinor: 4399, incomeMinor: 120000, missingRates: 1 })
  })
  it('пустой день', () => {
    expect(daySummary([])).toEqual({ spentMinor: 0, incomeMinor: 0, missingRates: 0 })
  })
})

describe('guessCategory', () => {
  const spend = (id: number, categoryId: number | null, note = '', kind: Entry['kind'] = 'expense'): Entry => ({
    id, kind, date: '2026-10-12', amountMinor: -100, currency: 'GEL', baseMinor: -100, accountId: 1, toAccountId: null, categoryId, note, inMinor: null, inCurrency: null, feeMinor: 0,
  })
  // Новые первыми: продукты 3 раза, кафе 2, такси 2 (но свежее кафе), доход и запись без категории не считаются.
  const entries = [spend(1, 20), spend(2, 10, 'Кофе '), spend(3, 30), spend(4, null), spend(5, 10), spend(6, 30), spend(7, 20), spend(8, 20), spend(9, 99, '', 'income')]

  it('guesses the category from the latest expense with the same note', () => {
    expect(guessCategory(entries, 'кофе')).toBe(10)
    expect(guessCategory(entries, 'такси')).toBeNull()
    expect(guessCategory(entries, '  ')).toBeNull()
  })
})

describe('recordStreak', () => {
  const at = (date: string): Entry => ({ id: 1, kind: 'expense', date, amountMinor: -100, currency: 'GEL', baseMinor: -100, accountId: 1, toAccountId: null, categoryId: 1, note: '', inMinor: null, inCurrency: null, feeMinor: 0 })

  it('counts consecutive days including today', () => {
    expect(recordStreak([at('2026-10-12'), at('2026-10-11'), at('2026-10-11'), at('2026-10-10'), at('2026-10-08')], '2026-10-12')).toBe(3)
  })

  it('keeps the streak alive in the morning before the first record', () => {
    expect(recordStreak([at('2026-10-11'), at('2026-10-10')], '2026-10-12')).toBe(2)
  })

  it('ignores openings and transfers', () => {
    const other = (kind: Entry['kind'], date: string): Entry => ({ ...at(date), kind })
    expect(recordStreak([other('opening', '2026-10-12'), other('transfer', '2026-10-11')], '2026-10-12')).toBe(0)
  })

  it('is zero after a gap', () => {
    expect(recordStreak([at('2026-10-09')], '2026-10-12')).toBe(0)
    expect(recordStreak([], '2026-10-12')).toBe(0)
  })
})

describe('categoryTotals', () => {
  const e = (date: string, baseMinor: number | null, kind: Entry['kind'] = 'expense'): Entry => ({ id: 1, kind, date, amountMinor: baseMinor ?? -1, currency: 'GEL', baseMinor, accountId: 1, toAccountId: null, categoryId: 1, note: '', inMinor: null, inCurrency: null, feeMinor: 0 })

  it('splits spending into this month and all time, skipping entries without a rate', () => {
    const list = [e('2026-10-05', -9000), e('2026-10-12', -500), e('2026-09-30', -90000), e('2026-10-06', null), e('2026-10-07', 100, 'income')]
    expect(categoryTotals(list, '2026-10-13')).toEqual({ month: 9500, all: 99500 })
  })
})
