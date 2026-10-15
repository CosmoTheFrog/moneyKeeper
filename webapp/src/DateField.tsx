import { useState } from 'react'
import { formatDayLong, monthGrid, monthTitle, shiftMonth } from './calendar'
import { t } from './i18n'
import { dayIn } from './ledger'
import { Modal } from './Modal'

type Props = {
  label: string
  // Пустая строка значит «не задана».
  value: string
  onChange: (day: string) => void
  placeholder: string
  // Можно ли очистить дату. У обязательных полей нет.
  clearable?: boolean
}

function CalendarIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="3.5" y="5" width="17" height="15" rx="3" />
      <path d="M3.5 10h17M8 3v4M16 3v4" />
    </svg>
  )
}

function Chevron({ left }: { left: boolean }) {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d={left ? 'M15 6l-6 6 6 6' : 'M9 6l6 6-6 6'} />
    </svg>
  )
}

// Поле даты: нажатие в любое место открывает календарь, выбор дня сразу закрывает его.
export function DateField({ label, value, onChange, placeholder, clearable = true }: Props) {
  const [open, setOpen] = useState(false)

  return (
    <div className="field">
      <span className="label">{label}</span>
      <div className="date-row">
        <button type="button" className="select" onClick={() => setOpen(true)}>
          <span className={value === '' ? 'select-empty' : ''}>{value === '' ? placeholder : formatDayLong(value)}</span>
          <CalendarIcon />
        </button>
        {clearable && value !== '' && (
          <button type="button" className="date-clear" aria-label={t.calendar.clear} onClick={() => onChange('')}>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.6" strokeLinecap="round" aria-hidden="true">
              <path d="M6 6l12 12M18 6L6 18" />
            </svg>
          </button>
        )}
      </div>
      {open && (
        <DayPicker
          title={label}
          value={value}
          onPick={(d) => {
            onChange(d)
            setOpen(false)
          }}
          onClose={() => setOpen(false)}
        />
      )}
    </div>
  )
}

type PickerProps = { title: string; value: string; onPick: (day: string) => void; onClose: () => void }

function DayPicker({ title, value, onPick, onClose }: PickerProps) {
  const today = dayIn(Intl.DateTimeFormat().resolvedOptions().timeZone)
  const [month, setMonth] = useState((value === '' ? today : value).slice(0, 7))

  return (
    <Modal title={title} onClose={onClose}>
      <div className="cal-head">
        <button type="button" className="plus" aria-label={t.calendar.prev} onClick={() => setMonth(shiftMonth(month, -1))}>
          <Chevron left />
        </button>
        <span className="cal-month">{monthTitle(month)}</span>
        <button type="button" className="plus" aria-label={t.calendar.next} onClick={() => setMonth(shiftMonth(month, 1))}>
          <Chevron left={false} />
        </button>
      </div>
      <div className="cal-grid cal-week" aria-hidden="true">
        {t.calendar.weekdays.map((w) => (
          <span key={w}>{w}</span>
        ))}
      </div>
      <div className="cal-grid cal-single">
        {monthGrid(month).map((c) => {
          const classes = ['cal-day']
          if (!c.inMonth) classes.push('cal-out')
          if (c.day === value) classes.push('cal-edge')
          if (c.day === today) classes.push('cal-today')
          return (
            <button key={c.day} type="button" className={classes.join(' ')} aria-pressed={c.day === value} aria-label={formatDayLong(c.day)} onClick={() => onPick(c.day)}>
              {Number(c.day.slice(8))}
            </button>
          )
        })}
      </div>
      <button type="button" className="chip cal-today-btn" onClick={() => onPick(today)}>
        {t.calendar.today}
      </button>
    </Modal>
  )
}