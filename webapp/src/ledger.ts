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
  for (const t of txs) {
    if (t.transfer_id === null) continue
    if (PAIRED.includes(t.kind) && t.amount_minor > 0) incoming.set(t.transfer_id, t)
    if (t.kind === 'fee') fees.set(t.transfer_id, (fees.get(t.transfer_id) ?? 0) - t.amount_minor)
  }
  const out: Entry[] = []
  for (const t of txs) {
    if (t.kind === 'fee') continue
    const paired = PAIRED.includes(t.kind)
    if (paired && t.amount_minor > 0 && txs.some((o) => o.transfer_id === t.transfer_id && o.amount_minor < 0 && PAIRED.includes(o.kind))) continue
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
