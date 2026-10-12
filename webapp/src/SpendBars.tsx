import { formatRange, monthTitle } from './calendar'
import { t } from './i18n'
import { Money } from './MoneyText'
import type { Buckets } from './report'

type Props = {
  buckets: Buckets
  currency: string
  today: string
  // Выбранная корзина: по ней фильтруется остальной экран. Повторное нажатие снимает выбор.
  selected: string | null
  onSelect: (key: string | null) => void
}

function label(unit: Buckets['unit'], key: string, today: string): string {
  return unit === 'month' ? monthTitle(key) : formatRange({ from: key, to: key }, today)
}

// Столбики трат по дням или месяцам. Нажатие на столбик выбирает этот день или месяц.
export function SpendBars({ buckets, currency, today, selected, onSelect }: Props) {
  const { unit, items } = buckets
  const max = Math.max(...items.map((b) => b.totalMinor), 0)
  const peak = items.find((b) => b.totalMinor === max && max > 0)
  const current = items.find((b) => b.key === selected) ?? peak
  const first = items[0]
  const last = items[items.length - 1]

  return (
    <div className="card bars-card">
      <div className="bars" role="group" aria-label={t.accountReport.when}>
        {items.map((b) => (
          <button
            key={b.key}
            type="button"
            className={b.key === selected ? 'bar bar-on' : 'bar'}
            aria-pressed={b.key === selected}
            aria-label={label(unit, b.key, today)}
            onClick={() => onSelect(b.key === selected ? null : b.key)}
          >
            <span className="bar-fill" style={{ height: max === 0 ? '0%' : `${Math.max((b.totalMinor * 100) / max, b.totalMinor > 0 ? 4 : 0)}%` }} />
          </button>
        ))}
      </div>
      {first !== undefined && last !== undefined && (
        <div className="bars-axis">
          <span>{unit === 'month' ? t.calendar.monthsShort[Number(first.key.slice(5)) - 1] : Number(first.key.slice(8))}</span>
          <span>{unit === 'month' ? t.calendar.monthsShort[Number(last.key.slice(5)) - 1] : Number(last.key.slice(8))}</span>
        </div>
      )}
      {current !== undefined && (
        <div className="row bars-pick">
          <span className="bars-pick-label">{label(unit, current.key, today)}</span>
          <Money minor={current.totalMinor} currency={currency} />
        </div>
      )}
    </div>
  )
}
