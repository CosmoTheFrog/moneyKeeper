import { useState } from 'react'
import { formatRange, monthGrid, monthTitle, pickDay, shiftMonth, type Selection } from './calendar'
import { t } from './i18n'
import { Modal } from './Modal'
import type { Range } from './report'

type Props = {
  initial: Range
  today: string
  onApply: (range: Range) => void
  onClose: () => void
}

function Chevron({ left }: { left: boolean }) {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d={left ? 'M15 6l-6 6 6 6' : 'M9 6l6 6-6 6'} />
    </svg>
  )
}

// Календарь выбора периода: нажать первый день, затем последний. Будущие дни недоступны.
export function RangeCalendar({ initial, today, onApply, onClose }: Props) {
  const [sel, setSel] = useState<Selection>({ from: initial.from, to: initial.to })
  const [month, setMonth] = useState(initial.to.slice(0, 7))

  const last = sel.to ?? sel.from
  const todayMonth = today.slice(0, 7)

  return (
    <Modal title={t.calendar.title} onClose={onClose}>
      <p className="cal-caption">{sel.to === null ? t.calendar.pickEnd : formatRange({ from: sel.from, to: sel.to }, today)}</p>

      <div className="cal-head">
        <button type="button" className="plus" aria-label={t.calendar.prev} onClick={() => setMonth(shiftMonth(month, -1))}>
          <Chevron left />
        </button>
        <span className="cal-month">{monthTitle(month)}</span>
        <button type="button" className="plus" aria-label={t.calendar.next} disabled={month >= todayMonth} onClick={() => setMonth(shiftMonth(month, 1))}>
          <Chevron left={false} />
        </button>
      </div>

      <div className="cal-grid cal-week" aria-hidden="true">
        {t.calendar.weekdays.map((w) => (
          <span key={w}>{w}</span>
        ))}
      </div>
      <div className="cal-grid">
        {monthGrid(month).map((c) => {
          const isEdge = c.day === sel.from || c.day === last
          const inside = c.day > sel.from && c.day < last
          const classes = ['cal-day']
          if (!c.inMonth) classes.push('cal-out')
          if (inside) classes.push('cal-in')
          if (isEdge) classes.push('cal-edge')
          if (c.day === today) classes.push('cal-today')
          return (
            <button
              key={c.day}
              type="button"
              className={classes.join(' ')}
              disabled={c.day > today}
              aria-pressed={isEdge}
              aria-label={formatRange({ from: c.day, to: c.day }, today)}
              onClick={() => setSel(pickDay(sel, c.day))}
            >
              {Number(c.day.slice(8))}
            </button>
          )
        })}
      </div>

      <button type="button" className="primary cal-apply" onClick={() => onApply({ from: sel.from, to: last })}>
        {t.calendar.apply}
      </button>
    </Modal>
  )
}
