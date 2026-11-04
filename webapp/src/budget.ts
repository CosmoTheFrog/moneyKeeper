import type { BudgetMonth, BudgetRow } from './api'
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

export type CompareRow = {
  categoryId: number
  budgetedNow: number
  spentNow: number
  budgetedThen: number
  spentThen: number
  // Потрачено сейчас минус потрачено тогда: больше нуля значит выросло.
  spentDelta: number
}

// compareMonths ставит месяц рядом с другим по категориям. Категории без сумм в обоих месяцах не попадают в список,
// сначала идут те, где трата изменилась сильнее всего.
export function compareMonths(now: BudgetMonth, then: BudgetMonth): CompareRow[] {
  const before = new Map(then.categories.map((r) => [r.category_id, r]))
  const rows: CompareRow[] = []
  for (const r of now.categories) {
    const b = before.get(r.category_id)
    const row = {
      categoryId: r.category_id,
      budgetedNow: r.budgeted_minor,
      spentNow: r.spent_minor,
      budgetedThen: b?.budgeted_minor ?? 0,
      spentThen: b?.spent_minor ?? 0,
      spentDelta: r.spent_minor - (b?.spent_minor ?? 0),
    }
    if (row.budgetedNow !== 0 || row.spentNow !== 0 || row.budgetedThen !== 0 || row.spentThen !== 0) rows.push(row)
  }
  return rows.sort((x, y) => Math.abs(y.spentDelta) - Math.abs(x.spentDelta))
}

export type Suggestion = { categoryId: number; minor: number }

// suggestBudget предлагает по каждой категории средние траты за прошлые месяцы. Месяцы, в которых не потрачено
// ничего, не считаются: у нового пользователя история короче трёх месяцев. Категории без трат не попадают в список.
export function suggestBudget(months: BudgetMonth[]): Suggestion[] {
  const active = months.filter((m) => m.categories.some((r) => r.spent_minor > 0))
  if (active.length === 0) return []
  const total = new Map<number, number>()
  for (const m of active) {
    for (const r of m.categories) total.set(r.category_id, (total.get(r.category_id) ?? 0) + Math.max(r.spent_minor, 0))
  }
  const out: Suggestion[] = []
  for (const [categoryId, sum] of total) {
    const minor = Math.round(sum / active.length)
    if (minor > 0) out.push({ categoryId, minor })
  }
  return out
}

export type SpendRow = {
  categoryId: number
  spentMinor: number
  // Трата этой же категории в прошлом месяце.
  previousMinor: number
}

export type Spending = {
  rows: SpendRow[]
  totalMinor: number
  previousTotalMinor: number
}

// spendingOf собирает траты месяца по категориям рядом с прошлым месяцем. В список попадают категории с тратой,
// самые большие первыми; итог и итог прошлого месяца считаются по всем категориям. Возврат средств не уменьшает трату.
export function spendingOf(now: BudgetMonth, then: BudgetMonth | null): Spending {
  const before = new Map((then?.categories ?? []).map((r) => [r.category_id, Math.max(r.spent_minor, 0)]))
  const rows = now.categories
    .filter((r) => r.spent_minor > 0)
    .map((r) => ({ categoryId: r.category_id, spentMinor: r.spent_minor, previousMinor: before.get(r.category_id) ?? 0 }))
    .sort((a, b) => b.spentMinor - a.spentMinor || a.categoryId - b.categoryId)
  return {
    rows,
    totalMinor: rows.reduce((s, r) => s + r.spentMinor, 0),
    previousTotalMinor: [...before.values()].reduce((s, v) => s + v, 0),
  }
}
