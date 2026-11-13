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

const RECENT_EXPENSES = 300

// frequentCategories: категории последних расходов, чаще используемые первыми, при равенстве выше та, что была недавно.
// entries идут от новых к старым, как их отдаёт сервер.
export function frequentCategories(entries: Entry[], limit: number): number[] {
  const seen = new Map<number, { count: number; first: number }>()
  let index = 0
  for (const e of entries) {
    if (e.kind !== 'expense' || e.categoryId === null) continue
    if (index++ >= RECENT_EXPENSES) break
    const s = seen.get(e.categoryId)
    if (s === undefined) seen.set(e.categoryId, { count: 1, first: index })
    else s.count++
  }
  return [...seen]
    .sort((a, b) => b[1].count - a[1].count || a[1].first - b[1].first)
    .slice(0, limit)
    .map(([id]) => id)
}

// guessCategory: категория последнего расхода с такой же заметкой (без учёта регистра и лишних пробелов), иначе null.
export function guessCategory(entries: Entry[], note: string): number | null {
  const key = note.trim().toLowerCase()
  if (key === '') return null
  for (const e of entries) {
    if (e.kind === 'expense' && e.categoryId !== null && e.note.trim().toLowerCase() === key) return e.categoryId
  }
  return null
}

// recordStreak: сколько дней подряд до today включительно есть хоть одна запись. Если сегодня записей нет, считается
// от вчера: день ещё не закончился, и серия не должна обнуляться с утра.
export function recordStreak(entries: Entry[], today: string): number {
  const days = new Set(entries.map((e) => e.date))
  let day = days.has(today) ? today : shiftDay(today, -1)
  let n = 0
  while (days.has(day)) {
    n++
    day = shiftDay(day, -1)
  }
  return n
}
