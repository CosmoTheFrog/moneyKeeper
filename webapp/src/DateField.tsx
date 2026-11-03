import { useState } from 'react'
import { formatDayLong, monthGrid, monthTitle, shiftMonth } from './calendar'
import { t } from './i18n'
import { MonthNav } from './MonthNav'
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
      <MonthNav variant="calendar" title={monthTitle(month)} onPrev={() => setMonth(shiftMonth(month, -1))} onNext={() => setMonth(shiftMonth(month, 1))} />
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