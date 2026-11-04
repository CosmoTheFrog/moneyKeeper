import type { BudgetRow } from './api'
import { parseAmount } from './money'

export type Tone = 'open' | 'done' | 'over'

// toBudgetTone: больше нуля осталось распределить, ноль всё распределено, меньше нуля заложено больше, чем есть.
export function toBudgetTone(toBudgetMinor: number): Tone {
  if (toBudgetMinor > 0) return 'open'
  return toBudgetMinor === 0 ? 'done' : 'over'
}

// fillPercent: какая часть заложенного потрачена, для полоски. Потрачено сверх заложенного это целая полоска;
// потрачено без заложенного тоже.
export function fillPercent(spentMinor: number, budgetedMinor: number): number {
  if (spentMinor <= 0) return 0
  if (budgetedMinor <= 0) return 100
  return Math.min(100, Math.round((spentMinor * 100) / budgetedMinor))
}

export type RowStatus = 'ok' | 'warn' | 'over' | 'noplan' | 'done'

const WARN_FILL = 92 // потрачено столько процентов заложенного: пора предупредить
const PACE_SLACK = 15 // на столько пунктов тратим быстрее календаря

// monthPace: какая часть месяца прошла к сегодняшнему дню, в процентах. Темп есть только у текущего месяца.
export function monthPace(month: string, today: string): number | null {
  if (today.slice(0, 7) !== month) return null
  const [y, m] = month.split('-').map(Number)
  const days = new Date(y ?? 1970, m ?? 1, 0).getDate()
  return (Number(today.slice(8, 10)) / days) * 100
}

// rowStatus: перерасход, траты без плана, всё потрачено, опережение темпа или всё в норме.
export function rowStatus(r: BudgetRow, pace: number | null): RowStatus {
  if (r.balance_minor < 0) return r.budgeted_minor <= 0 ? 'noplan' : 'over'
  if (r.spent_minor > 0 && r.balance_minor === 0) return 'done'
  const used = fillPercent(r.spent_minor, r.budgeted_minor)
  if (r.budgeted_minor > 0 && (used >= WARN_FILL || (pace !== null && used > pace + PACE_SLACK))) return 'warn'
  return 'ok'
}

// parseBudgeted разбирает сумму из поля: пусто и ноль значат «ничего не закладывать». Мусор и лишние знаки дают null.
export function parseBudgeted(input: string, currency: string): number | null {
  const text = input.trim()
  if (text === '' || /^0+([.,]0*)?$/.test(text)) return 0
  return parseAmount(text, currency)
}
