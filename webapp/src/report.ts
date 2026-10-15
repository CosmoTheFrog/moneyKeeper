import type { Category, CategoryGroup } from './api'
import type { Entry } from './ledger'

// Диапазон дат включительно, обе границы в виде YYYY-MM-DD. Строки этого вида сравниваются как даты.
export type Range = { from: string; to: string }

export type PresetPeriod = 'week' | 'twoWeeks' | 'threeWeeks' | 'month'

export function addDays(day: string, n: number): string {
  const [y, m, d] = day.split('-').map(Number)
  return new Date(Date.UTC(y ?? 1970, (m ?? 1) - 1, (d ?? 1) + n)).toISOString().slice(0, 10)
}

// Число дней от a до b, b позже a на один день даёт 1.
export function diffDays(a: string, b: string): number {
  const ms = (day: string) => {
    const [y, m, d] = day.split('-').map(Number)
    return Date.UTC(y ?? 1970, (m ?? 1) - 1, d ?? 1)
  }
  return Math.round((ms(b) - ms(a)) / 86_400_000)
}

// Неделя, две и три недели считаются назад от сегодня, месяц это текущий календарный месяц.
export function rangeFor(period: PresetPeriod, today: string): Range {
  switch (period) {
    case 'week':
      return { from: addDays(today, -6), to: today }
    case 'twoWeeks':
      return { from: addDays(today, -13), to: today }
    case 'threeWeeks':
      return { from: addDays(today, -20), to: today }
    case 'month':
      return { from: `${today.slice(0, 8)}01`, to: today }
  }
}

// allTimeRange тянется от первой траты до сегодня. Без трат это один сегодняшний день.
export function allTimeRange(entries: Entry[], today: string): Range {
  let from = today
  for (const e of entries) if (e.kind === 'expense' && e.date < from) from = e.date
  return { from, to: today }
}

// Period это выбор на экране: готовый период, всё время или свой диапазон из календаря.
export type Period = PresetPeriod | 'all' | 'custom'

export function resolveRange(period: Period, custom: Range, today: string, entries: Entry[]): Range {
  switch (period) {
    case 'custom':
      return custom
    case 'all':
      return allTimeRange(entries, today)
    default:
      return rangeFor(period, today)
  }
}

// Мера, в которой считаются траты: всё в базовой валюте по курсу дня записи или одна валюта как есть.
export type Measure = { kind: 'base' } | { kind: 'currency'; code: string }

// spendOf возвращает трату записи положительным числом. null: это не расход, другая валюта или у записи нет курса.
export function spendOf(e: Entry, measure: Measure): number | null {
  if (e.kind !== 'expense') return null
  if (measure.kind === 'base') return e.baseMinor === null ? null : -e.baseMinor
  return e.currency === measure.code ? -e.amountMinor : null
}

function inRange(e: Entry, range: Range): boolean {
  return e.date >= range.from && e.date <= range.to
}

export type Slice = {
  key: string
  label: string
  totalMinor: number
  groupId: number
}

export type CurrencyTotal = {
  currency: string
  totalMinor: number
}

// spendTotals считает расходы за период отдельно по каждой валюте как есть, без пересчёта.
// Первой идёт валюта first (если в ней были траты), остальные по сумме.
export function spendTotals(entries: Entry[], range: Range, first: string): CurrencyTotal[] {
  const totals = new Map<string, number>()
  for (const e of entries) {
    if (e.kind !== 'expense' || !inRange(e, range)) continue
    totals.set(e.currency, (totals.get(e.currency) ?? 0) - e.amountMinor)
  }
  return [...totals]
    .map(([currency, totalMinor]) => ({ currency, totalMinor }))
    .sort((a, b) => Number(b.currency === first) - Number(a.currency === first) || b.totalMinor - a.totalMinor || a.currency.localeCompare(b.currency))
}

// missingRates считает траты периода, у которых нет курса: в сумму «по курсу» они не входят.
export function missingRates(entries: Entry[], range: Range): number {
  return entries.filter((e) => e.kind === 'expense' && inRange(e, range) && e.baseMinor === null).length
}

// spendByCategory суммирует расходы за период в выбранной мере, самые большие первыми.
export function spendByCategory(entries: Entry[], categories: Category[], range: Range, measure: Measure): Slice[] {
  const byId = new Map(categories.map((c) => [c.id, c]))
  const totals = new Map<number, number>()
  for (const e of entries) {
    if (e.categoryId === null || !inRange(e, range)) continue
    const spend = spendOf(e, measure)
    if (spend !== null) totals.set(e.categoryId, (totals.get(e.categoryId) ?? 0) + spend)
  }

  const out: Slice[] = []
  for (const [id, totalMinor] of totals) {
    const c = byId.get(id)
    if (c !== undefined) out.push({ key: String(id), label: c.name, totalMinor, groupId: c.group_id })
  }
  return out.sort((a, b) => b.totalMinor - a.totalMinor || a.label.localeCompare(b.label, 'ru'))
}

export function spendByGroup(slices: Slice[], groups: CategoryGroup[]): Slice[] {
  const totals = new Map<number, number>()
  for (const s of slices) totals.set(s.groupId, (totals.get(s.groupId) ?? 0) + s.totalMinor)
  return groups
    .filter((g) => totals.has(g.id))
    .map((g) => ({ key: String(g.id), label: g.name, totalMinor: totals.get(g.id) ?? 0, groupId: g.id }))
    .sort((a, b) => b.totalMinor - a.totalMinor)
}

// spendEntries возвращает траты, из которых сложен кусок кольца: целой группы или одной категории.
// Берутся те же записи, что вошли в сумму (период, мера), новые сверху.
export function spendEntries(entries: Entry[], categories: Category[], range: Range, measure: Measure, slice: Slice, whole: 'group' | 'category'): Entry[] {
  const groupOf = new Map(categories.map((c) => [c.id, c.group_id]))
  return entries
    .filter((e) => {
      if (e.categoryId === null || !inRange(e, range) || spendOf(e, measure) === null) return false
      return whole === 'group' ? groupOf.get(e.categoryId) === slice.groupId : String(e.categoryId) === slice.key
    })
    .sort((a, b) => (a.date < b.date ? 1 : a.date > b.date ? -1 : b.id - a.id))
}

// inGroup оставляет категории одной группы, порядок (от большей суммы к меньшей) сохраняется.
export function inGroup(slices: Slice[], groupId: number): Slice[] {
  return slices.filter((s) => s.groupId === groupId)
}

// Корзина графика «когда»: день или месяц с суммой трат. from и to обрезаны по периоду.
export type Bucket = { key: string; from: string; to: string; totalMinor: number }

export type Buckets = { unit: 'day' | 'month'; items: Bucket[] }

const MAX_DAILY_BUCKETS = 31

// spendBuckets раскладывает траты по дням, если период не длиннее месяца, иначе по месяцам. Пустые корзины остаются.
export function spendBuckets(entries: Entry[], range: Range, measure: Measure): Buckets {
  const days = diffDays(range.from, range.to) + 1
  const items: Bucket[] = []

  if (days <= MAX_DAILY_BUCKETS) {
    for (let i = 0; i < days; i++) {
      const day = addDays(range.from, i)
      items.push({ key: day, from: day, to: day, totalMinor: 0 })
    }
    return { unit: 'day', items: fill(entries, range, measure, items) }
  }

  for (let ym = range.from.slice(0, 7); ym <= range.to.slice(0, 7); ym = nextMonth(ym)) {
    const first = `${ym}-01`
    const last = addDays(`${nextMonth(ym)}-01`, -1)
    items.push({ key: ym, from: first < range.from ? range.from : first, to: last > range.to ? range.to : last, totalMinor: 0 })
  }
  return { unit: 'month', items: fill(entries, range, measure, items) }
}

function fill(entries: Entry[], range: Range, measure: Measure, items: Bucket[]): Bucket[] {
  for (const e of entries) {
    if (!inRange(e, range)) continue
    const spend = spendOf(e, measure)
    const bucket = spend === null ? undefined : items.find((b) => e.date >= b.from && e.date <= b.to)
    if (bucket !== undefined && spend !== null) bucket.totalMinor += spend
  }
  return items
}

function nextMonth(ym: string): string {
  const [y, m] = ym.split('-').map(Number)
  return new Date(Date.UTC(y ?? 1970, m ?? 1, 1)).toISOString().slice(0, 7)
}

export function percent(part: number, total: number): number {
  return total === 0 ? 0 : Math.round((part * 100) / total)
}

export const RADIUS = 70
export const CIRCLE = 2 * Math.PI * RADIUS

export type Arc = { length: number; offset: number }

// arcs считает дуги кольца: длина пропорциональна доле, смещение накапливается со знаком минус.
// Это только для рисования, суммы остаются целыми.
export function arcs(values: number[]): Arc[] {
  const total = values.reduce((a, b) => a + b, 0)
  if (total === 0) return []
  let used = 0
  return values.map((v) => {
    const length = (v / total) * CIRCLE
    const arc = { length, offset: -used }
    used += length
    return arc
  })
}
