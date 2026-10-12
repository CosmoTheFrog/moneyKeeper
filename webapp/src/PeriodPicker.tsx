import { useState } from 'react'
import { daysInRange, formatRange } from './calendar'
import { plural, t } from './i18n'
import { RangeCalendar } from './RangeCalendar'
import type { Period, PresetPeriod, Range } from './report'

type Props = {
  // Готовые периоды в порядке показа; свой период (календарь) есть всегда.
  presets: readonly (PresetPeriod | 'all')[]
  period: Period
  // Период, который сейчас показан: по нему подпись под плашками.
  range: Range
  custom: Range
  today: string
  onPeriod: (period: Period) => void
  onCustom: (range: Range) => void
}

function CalendarIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="3.5" y="5" width="17" height="15" rx="3" />
      <path d="M3.5 10h17M8 3v4M16 3v4" />
    </svg>
  )
}

// Период одной строкой плашек и подписью, что именно показано. «Свой период» открывает календарь.
export function PeriodPicker({ presets, period, range, custom, today, onPeriod, onCustom }: Props) {
  const [calendar, setCalendar] = useState(false)
  const days = daysInRange(range)

  return (
    <div className="period">
      <div className="period-row" role="radiogroup" aria-label={t.calendar.title}>
        <div className="period-chips">
          {presets.map((p) => (
            <button key={p} type="button" role="radio" aria-checked={p === period} className={p === period ? 'pill pill-on' : 'pill'} onClick={() => onPeriod(p)}>
              {t.reports.periods[p]}
            </button>
          ))}
        </div>
        <button
          type="button"
          role="radio"
          aria-checked={period === 'custom'}
          aria-label={t.reports.periods.custom}
          className={period === 'custom' ? 'pill pill-on pill-icon' : 'pill pill-icon'}
          onClick={() => setCalendar(true)}
        >
          <CalendarIcon />
          {period === 'custom' && formatRange(custom, today)}
        </button>
      </div>
      <p className="period-caption">
        {formatRange(range, today)} · {days} {plural(days, t.calendar.days)}
      </p>

      {calendar && (
        <RangeCalendar
          initial={period === 'custom' ? custom : range}
          today={today}
          onClose={() => setCalendar(false)}
          onApply={(r) => {
            onCustom(r)
            setCalendar(false)
          }}
        />
      )}
    </div>
  )
}
