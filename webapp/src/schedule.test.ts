import { describe, expect, it } from 'vitest'
import type { Occurrence, Schedule } from './api'
import { amountParts, daysText, describeRepeat, groupOccurrences, parseDays, sectionOf } from './schedule'

const labels = { every: 'Раз в', month: 'Каждый месяц', months: 'мес.', last: 'последнего дня', and: 'и' }

function schedule(name: string, over: Partial<Schedule> = {}): Schedule {
  return {
    id: 1, name, category_id: 1, account_id: null, currency: 'GEL', amount_kind: 'exact', amount_minor: 100, amount_max_minor: null,
    every_months: 1, days: [15], start_month: '2026-09', weekend_shift: 'none', remind_before: 1, ...over,
  }
}

function occ(name: string, due: string, status: Occurrence['status']): Occurrence {
  return { schedule: schedule(name), due_on: due, status, transaction_id: null }
}

describe('parseDays', () => {
  it.each([
    ['15', false, [15]],
    ['1, 15', false, [1, 15]],
    ['15 1', false, [1, 15]],
    ['15;1,15', false, [1, 15]],
    ['', true, [0]],
    ['5', true, [0, 5]],
    ['31', false, [31]],
    ['  7  ', false, [7]],
  ])('%j last=%s → %j', (text, last, want) => {
    expect(parseDays(text, last)).toEqual(want)
  })

  it.each([
    ['', false],
    ['0', false],
    ['32', false],
    ['abc', false],
    ['1.5', false],
    ['1, x', true],
    ['1 2 3 4 5 6', false],
    ['1 2 3 4 5', true],
    ['-3', false],
    ['100', false],
  ])('%j last=%s → null', (text, last) => {
    expect(parseDays(text, last)).toBeNull()
  })
})

describe('daysText and describeRepeat', () => {
  it('writes the fields without the last day', () => {
    expect(daysText([0, 1, 15])).toBe('1, 15')
    expect(daysText([0])).toBe('')
  })

  it.each([
    [1, [15], 'Каждый месяц, 15-го'],
    [3, [1], 'Раз в 3 мес., 1-го'],
    [1, [0], 'Каждый месяц, последнего дня'],
    [1, [15, 1], 'Каждый месяц, 1-го и 15-го'],
    [2, [1, 15, 0], 'Раз в 2 мес., 1-го, 15-го и последнего дня'],
  ])('every %i %j → %s', (every, days, want) => {
    expect(describeRepeat(every, days, labels)).toBe(want)
  })
})

describe('sectionOf and groupOccurrences', () => {
  const today = '2026-10-12'

  it.each([
    ['overdue', '2026-10-01', 'overdue'],
    ['upcoming', '2026-10-12', 'soon'],
    ['upcoming', '2026-10-19', 'soon'],
    ['upcoming', '2026-10-20', 'later'],
    ['paid', '2026-10-05', 'closed'],
    ['skipped', '2026-10-25', 'closed'],
  ] as const)('%s on %s → %s', (status, due, want) => {
    expect(sectionOf({ status, due_on: due }, today)).toBe(want)
  })

  it('sorts inside the sections and puts the freshest closed first', () => {
    const g = groupOccurrences(
      [
        occ('Б', '2026-10-20', 'upcoming'),
        occ('А', '2026-10-20', 'upcoming'),
        occ('Аренда', '2026-10-15', 'upcoming'),
        occ('Интернет', '2026-09-30', 'overdue'),
        occ('Старый', '2026-09-15', 'paid'),
        occ('Свежий', '2026-10-05', 'paid'),
      ],
      today,
    )
    expect(g.overdue.map((o) => o.schedule.name)).toEqual(['Интернет'])
    expect(g.soon.map((o) => o.schedule.name)).toEqual(['Аренда'])
    expect(g.later.map((o) => o.schedule.name)).toEqual(['А', 'Б'])
    expect(g.closed.map((o) => o.schedule.name)).toEqual(['Свежий', 'Старый'])
  })

  it('does not mutate the input', () => {
    const list = [occ('Б', '2026-10-20', 'upcoming'), occ('А', '2026-10-15', 'upcoming')]
    groupOccurrences(list, today)
    expect(list.map((o) => o.schedule.name)).toEqual(['Б', 'А'])
  })
})

describe('amountParts', () => {
  it('shows an upper bound only for a range', () => {
    expect(amountParts({ amount_kind: 'range', amount_minor: 100, amount_max_minor: 300 })).toEqual({ kind: 'range', from: 100, to: 300 })
    expect(amountParts({ amount_kind: 'approx', amount_minor: 100, amount_max_minor: null })).toEqual({ kind: 'approx', from: 100, to: null })
  })
})
