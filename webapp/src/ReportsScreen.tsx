import { useMemo, useState } from 'react'
import type { Account, Category } from './api'
import { t } from './i18n'
import type { Entry } from './ledger'
import { Money } from './MoneyText'
import { ChipPicker } from './Pickers'
import { RADIUS, addDays, arcs, percent, rangeFor, spendByCategory, spendByGroup, type PresetPeriod, type Range, type Slice } from './report'

type Props = {
  entries: Entry[]
  categories: Category[]
  accounts: Account[]
  baseCurrency: string
  today: string
}

type Period = PresetPeriod | 'custom'
type By = 'groups' | 'categories'

const periodOptions = (['week', 'twoWeeks', 'threeWeeks', 'month', 'custom'] as const).map((p) => ({ value: p, label: t.reports.periods[p] }))
const byOptions = [
  { value: 'groups' as const, label: t.reports.byGroups },
  { value: 'categories' as const, label: t.reports.byCategories },
]

// Цвета групп; у категорий цвет берётся по кругу из этой же палитры.
const palette = ['#0e7c7b', '#4c5bd4', '#d9892b', '#d4577a', '#7a8794', '#5fb3b0', '#7c88e0', '#a9b1ec', '#cdd2f5', '#a5d3d1']

export function ReportsScreen({ entries, categories, accounts, baseCurrency, today }: Props) {
  const [period, setPeriod] = useState<Period>('month')
  const [custom, setCustom] = useState<Range>({ from: addDays(today, -29), to: today })
  const [by, setBy] = useState<By>('groups')
  const [currency, setCurrency] = useState(baseCurrency)

  const currencies = useMemo(() => [...new Set(accounts.map((a) => a.currency))], [accounts])
  const range = period === 'custom' ? custom : rangeFor(period, today)

  const slices = useMemo(() => spendByCategory(entries, categories, range, currency), [entries, categories, range, currency])
  const shown: Slice[] = by === 'groups' ? spendByGroup(slices, (g) => t.groups[g]) : slices
  const total = shown.reduce((s, x) => s + x.totalMinor, 0)
  const drawn = arcs(shown.map((s) => s.totalMinor))

  return (
    <main className="screen">
      <h1>{t.reports.title}</h1>

      <ChipPicker value={period} options={periodOptions} onChange={setPeriod} />
      {period === 'custom' && (
        <div className="card dates">
          <label className="field">
            <span className="label">{t.reports.from}</span>
            <input type="date" value={custom.from} max={custom.to} onChange={(e) => e.target.value !== '' && setCustom({ ...custom, from: e.target.value })} />
          </label>
          <label className="field">
            <span className="label">{t.reports.to}</span>
            <input type="date" value={custom.to} min={custom.from} onChange={(e) => e.target.value !== '' && setCustom({ ...custom, to: e.target.value })} />
          </label>
        </div>
      )}
      <ChipPicker value={by} options={byOptions} onChange={setBy} />
      {currencies.length > 1 && <ChipPicker value={currency} options={currencies.map((c) => ({ value: c, label: c }))} onChange={setCurrency} />}

      {shown.length === 0 ? (
        <p className="hint">{t.reports.empty}</p>
      ) : (
        <div className="card">
          <div className="donut">
            <svg viewBox="0 0 200 200" role="img" aria-label={t.reports.spent}>
              <g transform="rotate(-90 100 100)">
                <circle cx="100" cy="100" r={RADIUS} fill="none" style={{ stroke: 'var(--line)' }} strokeWidth="24" />
                {drawn.map((a, i) => (
                  <circle
                    key={shown[i]?.key}
                    cx="100"
                    cy="100"
                    r={RADIUS}
                    fill="none"
                    stroke={palette[i % palette.length]}
                    strokeWidth="24"
                    strokeDasharray={`${a.length} 1000`}
                    strokeDashoffset={a.offset}
                  />
                ))}
              </g>
            </svg>
            <div className="donut-center">
              <span className="donut-label">{t.reports.spent}</span>
              <Money minor={total} currency={currency} />
            </div>
          </div>

          <div className="legend">
            {shown.map((s, i) => (
              <div key={s.key} className="legend-row">
                <span className="dot" style={{ background: palette[i % palette.length] }} />
                <span className="legend-name">{s.label}</span>
                <Money minor={s.totalMinor} currency={currency} />
                <span className="legend-pct">{percent(s.totalMinor, total)}%</span>
              </div>
            ))}
          </div>
        </div>
      )}
      {currencies.length > 1 && <p className="hint tight">{t.reports.noRates}</p>}
    </main>
  )
}
