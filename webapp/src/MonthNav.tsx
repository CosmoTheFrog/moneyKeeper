import type { ReactNode } from 'react'
import { t } from './i18n'

export function Chevron({ left }: { left: boolean }) {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d={left ? 'M15 6l-6 6 6 6' : 'M9 6l6 6-6 6'} />
    </svg>
  )
}

type Props = {
  title: ReactNode
  onPrev: () => void
  onNext: () => void
  nextDisabled?: boolean
  // 'calendar' это шапка окна с календарём, 'page' шапка экрана.
  variant?: 'page' | 'calendar'
}

// Заголовок месяца или периода со стрелками назад и вперёд.
export function MonthNav({ title, onPrev, onNext, nextDisabled = false, variant = 'page' }: Props) {
  const calendar = variant === 'calendar'
  return (
    <div className={calendar ? 'cal-head' : 'month-nav'}>
      <button type="button" className="nav-arrow" aria-label={t.calendar.prev} onClick={onPrev}>
        <Chevron left />
      </button>
      <span className={calendar ? 'cal-month' : 'month-title'}>{title}</span>
      <button type="button" className="nav-arrow" aria-label={t.calendar.next} disabled={nextDisabled} onClick={onNext}>
        <Chevron left={false} />
      </button>
    </div>
  )
}
