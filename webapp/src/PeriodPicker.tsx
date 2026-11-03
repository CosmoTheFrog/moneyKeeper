import { daysInRange, formatRange, monthTitle } from './calendar'
import { plural, t } from './i18n'
import { MonthNav } from './MonthNav'
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
      <MonthNav
        title={
          <>
            {title}
            <span className="period-days">
              {days} {plural(days, t.calendar.days)}
            </span>
          </>
        }
        onPrev={() => onOffset(offset - 1)}
        onNext={() => onOffset(offset + 1)}
        nextDisabled={offset >= 0}
      />
    </div>
  )
}
