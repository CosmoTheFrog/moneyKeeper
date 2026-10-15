import { t } from './i18n'
import { addDays, diffDays, type Range } from './report'

// Выбор периода в календаре: from уже выбран, to ещё нет, пока человек не нажал второй день.
export type Selection = { from: string; to: string | null }

export type Cell = { day: string; inMonth: boolean }

// pickDay: первое нажатие начинает период, второе замыкает его, нажатие раньше начала меняет границы местами.
// Нажатие после готового периода начинает новый.
export function pickDay(sel: Selection, day: string): Selection {
  if (sel.to !== null) return { from: day, to: null }
  return day < sel.from ? { from: day, to: sel.from } : { from: sel.from, to: day }
}

// shiftMonth сдвигает месяц вида YYYY-MM на delta месяцев.
export function shiftMonth(ym: string, delta: number): string {
  const [y, m] = ym.split('-').map(Number)
  return new Date(Date.UTC(y ?? 1970, (m ?? 1) - 1 + delta, 1)).toISOString().slice(0, 7)
}

// monthGrid отдаёт шесть недель по семь дней, неделя с понедельника; дни соседних месяцев помечены inMonth: false.
export function monthGrid(ym: string): Cell[] {
  const first = `${ym}-01`
  const [y, m, d] = first.split('-').map(Number)
  const weekday = (new Date(Date.UTC(y ?? 1970, (m ?? 1) - 1, d ?? 1)).getUTCDay() + 6) % 7
  return Array.from({ length: 42 }, (_, i) => {
    const day = addDays(first, i - weekday)
    return { day, inMonth: day.slice(0, 7) === ym }
  })
}

export function monthTitle(ym: string): string {
  const [y, m] = ym.split('-').map(Number)
  return `${t.calendar.months[(m ?? 1) - 1] ?? ''} ${y ?? ''}`
}

// «6 окт», год добавляется, если он не текущий.
function formatDay(day: string, today: string, withMonth = true): string {
  const [y, m, d] = day.split('-').map(Number)
  const year = y === Number(today.slice(0, 4)) ? '' : ` ${y}`
  return withMonth ? `${d} ${t.calendar.monthsShort[(m ?? 1) - 1] ?? ''}${year}` : String(d)
}

// formatRange: «12 окт», «6 – 12 окт», «12 сен – 12 окт». Другой год дописывается.
export function formatRange(range: Range, today: string): string {
  if (range.from === range.to) return formatDay(range.from, today)
  const sameMonth = range.from.slice(0, 7) === range.to.slice(0, 7)
  const start = formatDay(range.from, today, !sameMonth)
  return `${start} – ${formatDay(range.to, today)}`
}

export function daysInRange(range: Range): number {
  return diffDays(range.from, range.to) + 1
}

// formatDayLong: «12 окт 2026», всегда с годом: для полей даты, где важно видеть всё.
export function formatDayLong(day: string): string {
  const [y, m, d] = day.split('-').map(Number)
  return `${d} ${t.calendar.monthsShort[(m ?? 1) - 1] ?? ''} ${y}`
}