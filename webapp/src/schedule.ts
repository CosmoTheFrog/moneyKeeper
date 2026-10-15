import type { Occurrence, Schedule } from './api'
import { diffDays } from './report'

const MAX_DAYS = 5

// parseDays разбирает «1, 15» и флаг «последний день месяца» в список чисел расписания (0 это последний день).
// null, если чисел нет, их больше пяти, число вне 1–31 или в тексте мусор.
export function parseDays(text: string, lastDay: boolean): number[] | null {
  const parts = text.split(/[\s,;]+/).filter((p) => p !== '')
  const days: number[] = []
  for (const p of parts) {
    if (!/^\d{1,2}$/.test(p)) return null
    const n = Number(p)
    if (n < 1 || n > 31) return null
    if (!days.includes(n)) days.push(n)
  }
  if (lastDay) days.push(0)
  days.sort((a, b) => a - b)
  return days.length === 0 || days.length > MAX_DAYS ? null : days
}

// daysText возвращает числа для поля ввода без «последнего дня»: [0, 1, 15] → «1, 15».
export function daysText(days: number[]): string {
  return days.filter((d) => d > 0).join(', ')
}

// describeRepeat: «Каждый месяц, 15-го», «Раз в 3 мес., 1-го и последнего дня», «Каждый год…» не выделяем, это раз в 12 мес.
export function describeRepeat(everyMonths: number, days: number[], labels: { every: string; month: string; months: string; last: string; and: string }): string {
  // Последний день месяца (0) называем в конце.
  const parts = [...days].sort((a, b) => (a || 32) - (b || 32)).map((d) => (d === 0 ? labels.last : `${d}-го`))
  const when = parts.length > 1 ? `${parts.slice(0, -1).join(', ')} ${labels.and} ${parts[parts.length - 1]}` : (parts[0] ?? '')
  const every = everyMonths === 1 ? labels.month : `${labels.every} ${everyMonths} ${labels.months}`
  return `${every}, ${when}`
}

export type Section = 'overdue' | 'soon' | 'later' | 'closed'

const SOON_DAYS = 7

// sectionOf раскладывает платёж: просроченные, ближайшая неделя, дальше, и закрытые (оплаченные и пропущенные).
export function sectionOf(o: Pick<Occurrence, 'status' | 'due_on'>, today: string): Section {
  if (o.status === 'paid' || o.status === 'skipped') return 'closed'
  if (o.status === 'overdue') return 'overdue'
  return diffDays(today, o.due_on) <= SOON_DAYS ? 'soon' : 'later'
}

// groupOccurrences делит платежи по разделам, внутри раздела по дате. Закрытые идут от свежих к старым.
export function groupOccurrences(list: Occurrence[], today: string): Record<Section, Occurrence[]> {
  const out: Record<Section, Occurrence[]> = { overdue: [], soon: [], later: [], closed: [] }
  for (const o of [...list].sort((a, b) => a.due_on.localeCompare(b.due_on) || a.schedule.name.localeCompare(b.schedule.name, 'ru'))) {
    out[sectionOf(o, today)].push(o)
  }
  out.closed.reverse()
  return out
}

// amountParts: что показать про сумму платежа: вид и числа. Для «примерно» это сумма, допуск пишется отдельно.
export function amountParts(s: Pick<Schedule, 'amount_kind' | 'amount_minor' | 'amount_max_minor'>): { kind: Schedule['amount_kind']; from: number; to: number | null } {
  return { kind: s.amount_kind, from: s.amount_minor, to: s.amount_kind === 'range' ? s.amount_max_minor : null }
}
