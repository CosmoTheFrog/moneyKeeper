import { useState } from 'react'
import type { Account, Category } from './api'
import { EntryList } from './EntryList'
import { categoryColors } from './categoryColor'
import { t } from './i18n'
import type { Entry } from './ledger'
import { Money } from './MoneyText'
import { RADIUS, arcs, CIRCLE, percent, spendEntries, type Measure, type Range, type Slice } from './report'

type Props = {
  // Траты по категориям за период, самые большие первыми.
  slices: Slice[]
  categories: Category[]
  accounts: Account[]
  entries: Entry[]
  range: Range
  measure: Measure
  currency: string
  today: string
  onOpen: (entry: Entry) => void
}

const POP = 6

// Кольцо и список трат по категориям. Нажатие на кусок выдвигает его и показывает траты, из которых он сложен.
export function SpendBreakdown({ slices, categories, accounts, entries, range, measure, currency, today, onOpen }: Props) {
  const [selected, setSelected] = useState<string | null>(null)

  const shown = slices
  const total = shown.reduce((s, x) => s + x.totalMinor, 0)
  const drawn = arcs(shown.map((s) => s.totalMinor))
  const current = shown.find((s) => s.key === selected)
  const currentIndex = current === undefined ? -1 : shown.indexOf(current)
  // Цвет куска это цвет его категории, такой же, как в операциях и бюджете.
  const hues = categoryColors(categories)
  const tone = (i: number): string => hues.get(Number(shown[i]?.key)) ?? 'var(--accent)'

  function pick(slice: Slice) {
    setSelected((now) => (now === slice.key ? null : slice.key))
  }

  function select(slice: Slice | undefined) {
    if (slice === undefined) return {}
    return {
      role: 'button',
      tabIndex: 0,
      'aria-pressed': slice.key === selected,
      onClick: () => pick(slice),
      onKeyDown: (e: { key: string; preventDefault: () => void }) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          pick(slice)
        }
      },
    }
  }

  // Сдвиг выбранного куска наружу по середине его дуги (в осях повёрнутой группы дуга начинается справа).
  function pop(i: number): string | undefined {
    const a = drawn[i]
    if (a === undefined || shown[i]?.key !== selected) return undefined
    const mid = ((-a.offset + a.length / 2) / CIRCLE) * 2 * Math.PI
    return `translate(${(Math.cos(mid) * POP).toFixed(2)}px, ${(Math.sin(mid) * POP).toFixed(2)}px)`
  }

  const list = current === undefined ? [] : spendEntries(entries, range, measure, current)

  return (
    <>
      {shown.length === 0 ? (
        <p className="hint">{t.reports.empty}</p>
      ) : (
        <div className="card">
          <div className="donut">
            <svg viewBox="-10 -10 220 220" role="img" aria-label={t.reports.spent}>
              <g transform="rotate(-90 100 100)">
                <circle cx="100" cy="100" r={RADIUS} fill="none" style={{ stroke: 'var(--line)' }} strokeWidth="24" />
                {drawn.map((a, i) => {
                  const slice = shown[i]
                  const active = slice?.key === selected
                  return (
                    <circle
                      key={slice?.key}
                      className="donut-seg donut-seg-tap"
                      cx="100"
                      cy="100"
                      r={RADIUS}
                      fill="none"
                      strokeWidth={active ? 30 : 24}
                      strokeDasharray={`${a.length} 1000`}
                      strokeDashoffset={a.offset}
                      style={{ stroke: tone(i), transform: pop(i), opacity: selected === null || active ? 1 : 0.35 }}
                      aria-label={slice?.label}
                      {...select(slice)}
                    />
                  )
                })}
              </g>
            </svg>
            <div className="donut-center">
              <span className="donut-label">{current === undefined ? t.reports.spent : current.label}</span>
              <Money minor={current === undefined ? total : current.totalMinor} currency={currency} />
              {current !== undefined && <span className="donut-pct">{percent(current.totalMinor, total)}%</span>}
            </div>
          </div>

          <div className="legend">
            {shown.map((s, i) => (
              <div key={s.key} className={s.key === selected ? 'legend-row legend-tap legend-on' : 'legend-row legend-tap'} {...select(s)}>
                <span className="dot" style={{ background: tone(i) }} />
                <span className="legend-name">{s.label}</span>
                <Money minor={s.totalMinor} currency={currency} />
                <span className="legend-pct">{percent(s.totalMinor, total)}%</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {current !== undefined && (
        <>
          <div className="row group-head">
            <span className="group-title" style={{ color: tone(currentIndex) }}>
              {current.label}
            </span>
          </div>
          <EntryList entries={list} accounts={accounts} categories={categories} today={today} onOpen={onOpen} noteFirst="withCategory" />
        </>
      )}
    </>
  )
}
