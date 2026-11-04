import type { Transaction, TransactionKind } from './api'

// Entry это строка списка: две ноги перевода или обмена склеены в одну. У перевода и обмена amountMinor это сколько ушло
// (положительное), inMinor и inCurrency сколько пришло, feeMinor комиссия обмена со счёта вне бюджета в валюте ушедшей суммы.
export type Entry = {
  id: number
  kind: TransactionKind
  date: string
  amountMinor: number
  currency: string
  // Сумма в базовой валюте по курсу дня записи, null если курса нет.
  baseMinor: number | null
  accountId: number
  toAccountId: number | null
  categoryId: number | null
  note: string
  inMinor: number | null
  inCurrency: string | null
  feeMinor: number
}
export type DayGroup = {
  date: string
  entries: Entry[]
}

// dayIn возвращает календарный день момента now в поясе timeZone как YYYY-MM-DD.
// «Сегодня» считается в поясе пользователя из его профиля, как и на сервере.
export function dayIn(timeZone: string, now: Date = new Date()): string {
  return new Intl.DateTimeFormat('en-CA', { timeZone, year: 'numeric', month: '2-digit', day: '2-digit' }).format(now)
}

const PAIRED: readonly string[] = ['transfer', 'exchange']

// toEntries склеивает ноги перевода и обмена. Если вторая нога не попала в выборку, показывается одна.
// Комиссия обмена со счёта вне бюджета (kind fee) входит в запись обмена и отдельной строкой не показывается.
export function toEntries(txs: Transaction[]): Entry[] {
  const incoming = new Map<number, Transaction>()
  const fees = new Map<number, number>()
  const hasOutgoing = new Set<number>()
  for (const t of txs) {
    if (t.transfer_id === null) continue
    if (PAIRED.includes(t.kind) && t.amount_minor > 0) incoming.set(t.transfer_id, t)
    if (PAIRED.includes(t.kind) && t.amount_minor < 0) hasOutgoing.add(t.transfer_id)
    if (t.kind === 'fee') fees.set(t.transfer_id, (fees.get(t.transfer_id) ?? 0) - t.amount_minor)
  }
  const out: Entry[] = []
  for (const t of txs) {
    if (t.kind === 'fee') continue
    const paired = PAIRED.includes(t.kind)
    if (paired && t.amount_minor > 0 && t.transfer_id !== null && hasOutgoing.has(t.transfer_id)) continue
    const to = paired && t.transfer_id !== null && t.amount_minor < 0 ? incoming.get(t.transfer_id) : undefined
    out.push({
      id: t.id,
      kind: t.kind,
      date: t.date,
      amountMinor: paired ? Math.abs(t.amount_minor) : t.amount_minor,
      currency: t.currency,
      baseMinor: t.base_minor !== null && paired ? Math.abs(t.base_minor) : t.base_minor,
      accountId: t.account_id,
      toAccountId: to?.account_id ?? null,
      categoryId: t.category_id,
      note: t.note,
      inMinor: to?.amount_minor ?? null,
      inCurrency: to?.currency ?? null,
      feeMinor: paired && t.transfer_id !== null && t.amount_minor < 0 ? (fees.get(t.transfer_id) ?? 0) : 0,
    })
  }
  return out
}
// groupByDay сохраняет порядок записей: сервер уже отдаёт их от новых к старым.
export function groupByDay(entries: Entry[]): DayGroup[] {
  const groups: DayGroup[] = []
  for (const e of entries) {
    const last = groups[groups.length - 1]
    if (last !== undefined && last.date === e.date) last.entries.push(e)
    else groups.push({ date: e.date, entries: [e] })
  }
  return groups
}

// shiftDay сдвигает календарный день YYYY-MM-DD на delta дней.
export function shiftDay(date: string, delta: number): string {
  const [y, m, d] = date.split('-').map(Number)
  return new Date(Date.UTC(y ?? 1970, (m ?? 1) - 1, (d ?? 1) + delta)).toISOString().slice(0, 10)
}

// dayNet: сумма доходов и расходов дня со знаком в валюте пересчёта. null, если таких записей нет или у какой-то нет курса:
// неполную сумму показывать нельзя. Переводы и обмены в итог не входят.
export function dayNet(entries: Entry[]): number | null {
  let sum = 0
  let any = false
  for (const e of entries) {
    if (e.kind !== 'expense' && e.kind !== 'income') continue
    if (e.baseMinor === null) return null
    sum += e.baseMinor
    any = true
  }
  return any ? sum : null
}

export type DaySpend = { date: string; spentMinor: number }

// lastDays: траты за последние n дней по дням, старые первыми, в валюте пересчёта. Расход без курса не входит.
export function lastDays(entries: Entry[], today: string, n: number): DaySpend[] {
  const days: DaySpend[] = Array.from({ length: n }, (_, i) => ({ date: shiftDay(today, i - (n - 1)), spentMinor: 0 }))
  const at = new Map(days.map((d) => [d.date, d]))
  for (const e of entries) {
    if (e.kind !== 'expense' || e.baseMinor === null) continue
    const day = at.get(e.date)
    if (day !== undefined) day.spentMinor += -e.baseMinor
  }
  return days
}

export type DayPart = { categoryId: number | null; totalMinor: number }

// dayParts: расходы дня по категориям, больше первыми, в валюте пересчёта. Расход без курса не входит.
export function dayParts(entries: Entry[]): DayPart[] {
  const total = new Map<number | null, number>()
  for (const e of entries) {
    if (e.kind !== 'expense' || e.baseMinor === null) continue
    total.set(e.categoryId, (total.get(e.categoryId) ?? 0) - e.baseMinor)
  }
  return [...total].map(([categoryId, totalMinor]) => ({ categoryId, totalMinor })).sort((a, b) => b.totalMinor - a.totalMinor)
}

// entryMatches: запись подходит под строку поиска, если она есть в заметке, названии категории или счёта.
export function entryMatches(e: Entry, query: string, accountName: Map<number, string>, categoryName: Map<number, string>): boolean {
  const q = query.trim().toLowerCase()
  if (q === '') return true
  const hay = [e.note, e.categoryId === null ? '' : (categoryName.get(e.categoryId) ?? ''), accountName.get(e.accountId) ?? '', e.toAccountId === null ? '' : (accountName.get(e.toAccountId) ?? '')]
  return hay.some((h) => h.toLowerCase().includes(q))
}

export type DaySummary = { spentMinor: number; incomeMinor: number; missingRates: number }

// daySummary: траты (положительное число) и доход дня в валюте пересчёта. Записи без курса не входят и считаются отдельно.
export function daySummary(entries: Entry[]): DaySummary {
  const out: DaySummary = { spentMinor: 0, incomeMinor: 0, missingRates: 0 }
  for (const e of entries) {
    if (e.kind !== 'expense' && e.kind !== 'income') continue
    if (e.baseMinor === null) {
      out.missingRates++
      continue
    }
    if (e.kind === 'expense') out.spentMinor += -e.baseMinor
    else out.incomeMinor += e.baseMinor
  }
  return out
}
