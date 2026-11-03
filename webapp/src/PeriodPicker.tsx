import { daysInRange, formatRange, monthTitle } from './calendar'
import { plural, t } from './i18n'
import type { Period, Range } from './report'

type Props = {
  period: Period
  // Сдвиг в неделях или месяцах от текущего: 0 текущий, -1 прошлый.
  offset: number
  // Период, который сейчас показан.
  range: Range
  today: string
  onPeriod: (period: Period) => void
  onOffset: (offset: number) => void
}

const periods = ['week', 'month'] as const

function Chevron({ left }: { left: boolean }) {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d={left ? 'M15 6l-6 6 6 6' : 'M9 6l6 6-6 6'} />
    </svg>
  )
}

// Неделя или месяц плашками, под ними стрелки на прошлый и следующий период. Дальше текущего листать нельзя.
export function PeriodPicker({ period, offset, range, today, onPeriod, onOffset }: Props) {
  const days = daysInRange(range)
  const title = period === 'month' ? monthTitle(range.from.slice(0, 7)) : formatRange(range, today)

  return (
    <div className="period">
      <div className="period-row" role="radiogroup" aria-label={t.reports.title}>
        {periods.map((p) => (
          <button key={p} type="button" role="radio" aria-checked={p === period} className={p === period ? 'pill pill-on' : 'pill'} onClick={() => onPeriod(p)}>
            {t.reports.periods[p]}
          </button>
        ))}
      </div>
      <div className="month-nav">
        <button type="button" className="plus" aria-label={t.calendar.prev} onClick={() => onOffset(offset - 1)}>
          <Chevron left />
        </button>
        <span className="month-title">
          {title}
          <span className="period-days">
            {days} {plural(days, t.calendar.days)}
          </span>
        </span>
        <button type="button" className="plus" aria-label={t.calendar.next} disabled={offset >= 0} onClick={() => onOffset(offset + 1)}>
          <Chevron left={false} />
        </button>
      </div>
    </div>
  )
}
