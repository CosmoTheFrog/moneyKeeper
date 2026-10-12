import type { Transaction, TransactionKind } from './api'

// Entry это строка списка: две ноги перевода склеены в одну.
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

// toEntries склеивает ноги перевода. Если вторая нога не попала в выборку, показывается одна.
export function toEntries(txs: Transaction[]): Entry[] {
  const incoming = new Map<number, Transaction>()
  for (const t of txs) {
    if (t.kind === 'transfer' && t.transfer_id !== null && t.amount_minor > 0) incoming.set(t.transfer_id, t)
  }
  const out: Entry[] = []
  for (const t of txs) {
    if (t.kind === 'transfer' && t.amount_minor > 0 && txs.some((o) => o.transfer_id === t.transfer_id && o.amount_minor < 0)) continue
    const to = t.kind === 'transfer' && t.transfer_id !== null && t.amount_minor < 0 ? incoming.get(t.transfer_id) : undefined
    out.push({
      id: t.id,
      kind: t.kind,
      date: t.date,
      amountMinor: t.kind === 'transfer' ? Math.abs(t.amount_minor) : t.amount_minor,
      currency: t.currency,
      baseMinor: t.base_minor !== null && t.kind === 'transfer' ? Math.abs(t.base_minor) : t.base_minor,
      accountId: t.account_id,
      toAccountId: to?.account_id ?? null,
      categoryId: t.category_id,
      note: t.note,
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
