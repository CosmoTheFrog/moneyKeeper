import { useMemo, useState } from 'react'
import type { Category, CategoryGroup } from './api'
import { t } from './i18n'
import type { Entry } from './ledger'
import { Money } from './MoneyText'
import { ChipPicker } from './Pickers'
import { RADIUS, addDays, arcs, inGroup, percent, rangeFor, spendByCategory, spendByGroup, spendTotals, type PresetPeriod, type Range, type Slice } from './report'

type Props = {
  entries: Entry[]
  groups: CategoryGroup[]
  categories: Category[]
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

export function ReportsScreen({ entries, groups, categories, baseCurrency, today }: Props) {
  const [period, setPeriod] = useState<Period>('month')
  const [custom, setCustom] = useState<Range>({ from: addDays(today, -29), to: today })
  const [by, setBy] = useState<By>('groups')
  const [currency, setCurrency] = useState(baseCurrency)
  // Открытая группа: кольцо показывает её категории, а не все группы сразу.
  const [openGroup, setOpenGroup] = useState<number | null>(null)

  const range = period === 'custom' ? custom : rangeFor(period, today)
  // Траты за период по валютам: пока нет курсов, они не складываются, но все видны.
  const totals = spendTotals(entries, range, baseCurrency)
  const currencies = totals.some((x) => x.currency === currency) ? totals.map((x) => x.currency) : [currency, ...totals.map((x) => x.currency)]
  const others = totals.filter((x) => x.currency !== currency)

  const slices = useMemo(() => spendByCategory(entries, categories, range, currency), [entries, categories, range, currency])
  const groupsView = by === 'groups' && openGroup === null
  const shown: Slice[] = openGroup !== null ? inGroup(slices, openGroup) : by === 'groups' ? spendByGroup(slices, groups) : slices
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
      <ChipPicker
        value={by}
        options={byOptions}
        onChange={(v) => {
          setBy(v)
          setOpenGroup(null)
        }}
      />
      {currencies.length > 1 && (
        <ChipPicker
          value={currency}
          options={currencies.map((c) => ({ value: c, label: c }))}
          onChange={(c) => {
            setCurrency(c)
            setOpenGroup(null)
          }}
        />
      )}

      {openGroup !== null && (
        <div className="row group-head">
          <button type="button" className="chip" onClick={() => setOpenGroup(null)}>
            {t.reports.allGroups}
          </button>
          <span className="group-title">{groups.find((g) => g.id === openGroup)?.name}</span>
        </div>
      )}

      {shown.length === 0 ? (
        <p className="hint">{t.reports.empty}</p>
      ) : (
        <div className="card">
          <div className="donut">
            <svg viewBox="0 0 200 200" role="img" aria-label={t.reports.spent}>
              <g transform="rotate(-90 100 100)">
                <circle cx="100" cy="100" r={RADIUS} fill="none" style={{ stroke: 'var(--line)' }} strokeWidth="24" />
                {drawn.map((a, i) => {
                  const slice = shown[i]
                  return (
                    <circle
                      key={slice?.key}
                      className={groupsView ? 'donut-seg donut-seg-tap' : 'donut-seg'}
                      cx="100"
                      cy="100"
                      r={RADIUS}
                      fill="none"
                      stroke={palette[i % palette.length]}
                      strokeWidth="24"
                      strokeDasharray={`${a.length} 1000`}
                      strokeDashoffset={a.offset}
                      role={groupsView ? 'button' : undefined}
                      tabIndex={groupsView ? 0 : undefined}
                      aria-label={groupsView ? slice?.label : undefined}
                      onClick={groupsView && slice !== undefined ? () => setOpenGroup(slice.groupId) : undefined}
                      onKeyDown={
                        groupsView && slice !== undefined
                          ? (e) => {
                              if (e.key === 'Enter' || e.key === ' ') {
                                e.preventDefault()
                                setOpenGroup(slice.groupId)
                              }
                            }
                          : undefined
                      }
                    />
                  )
                })}
              </g>
            </svg>
            <div className="donut-center">
              <span className="donut-label">{t.reports.spent}</span>
              <Money minor={total} currency={currency} />
            </div>
          </div>

          <div className="legend">
            {shown.map((s, i) => (
              <div
                key={s.key}
                className={groupsView ? 'legend-row legend-tap' : 'legend-row'}
                role={groupsView ? 'button' : undefined}
                tabIndex={groupsView ? 0 : undefined}
                onClick={groupsView ? () => setOpenGroup(s.groupId) : undefined}
                onKeyDown={
                  groupsView
                    ? (e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          setOpenGroup(s.groupId)
                        }
                      }
                    : undefined
                }
              >
                <span className="dot" style={{ background: palette[i % palette.length] }} />
                <span className="legend-name">{s.label}</span>
                <Money minor={s.totalMinor} currency={currency} />
                <span className="legend-pct">{percent(s.totalMinor, total)}%</span>
                {groupsView && (
                  <svg className="legend-chevron" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <path d="M9 6l6 6-6 6" />
                  </svg>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
      {others.length > 0 && (
        <div className="also">
          <span className="label">{t.reports.alsoSpent}</span>
          <div className="chips">
            {others.map((x) => (
              <div
                key={x.currency}
                className="pill pill-money"
                role="button"
                tabIndex={0}
                onClick={() => {
                  setCurrency(x.currency)
                  setOpenGroup(null)
                }}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault()
                    setCurrency(x.currency)
                    setOpenGroup(null)
                  }
                }}
              >
                <Money minor={x.totalMinor} currency={x.currency} />
              </div>
            ))}
          </div>
          <p className="hint tight">{t.reports.noRates}</p>
        </div>
      )}
    </main>
  )
}
