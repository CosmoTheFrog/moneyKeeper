import { useState } from 'react'
import type { CategoryGroup } from './api'
import { t } from './i18n'
import { Money } from './MoneyText'
import { Segmented } from './Pickers'
import { RADIUS, arcs, inGroup, percent, spendByGroup, type Slice } from './report'

type Props = {
  // Траты по категориям за период, самые большие первыми.
  slices: Slice[]
  groups: CategoryGroup[]
  currency: string
}

type By = 'groups' | 'categories'

const byOptions = [
  { value: 'groups' as const, label: t.reports.byGroups },
  { value: 'categories' as const, label: t.reports.byCategories },
]

// Цвета групп; у категорий цвет берётся по кругу из этой же палитры.
const palette = ['#0e7c7b', '#4c5bd4', '#d9892b', '#d4577a', '#7a8794', '#5fb3b0', '#7c88e0', '#a9b1ec', '#cdd2f5', '#a5d3d1']

function Chevron() {
  return (
    <svg className="legend-chevron" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M9 6l6 6-6 6" />
    </svg>
  )
}

// Кольцо и список трат: по группам или по категориям. Нажатие на группу открывает её категории.
export function SpendBreakdown({ slices, groups, currency }: Props) {
  const [by, setBy] = useState<By>('groups')
  // Открытая группа: кольцо показывает её категории, а не все группы сразу.
  const [openGroup, setOpenGroup] = useState<number | null>(null)

  const groupsView = by === 'groups' && openGroup === null
  const shown = openGroup !== null ? inGroup(slices, openGroup) : by === 'groups' ? spendByGroup(slices, groups) : slices
  const total = shown.reduce((s, x) => s + x.totalMinor, 0)
  const drawn = arcs(shown.map((s) => s.totalMinor))

  // Нажимаемое только в режиме групп: действие на мыши, клавиатуре и для скринридера.
  function tap(slice: Slice | undefined) {
    if (!groupsView || slice === undefined) return {}
    return {
      role: 'button',
      tabIndex: 0,
      onClick: () => setOpenGroup(slice.groupId),
      onKeyDown: (e: { key: string; preventDefault: () => void }) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          setOpenGroup(slice.groupId)
        }
      },
    }
  }

  return (
    <>
      <Segmented
        label={t.reports.byGroups}
        value={by}
        options={byOptions}
        onChange={(v) => {
          setBy(v)
          setOpenGroup(null)
        }}
      />

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
                      aria-label={groupsView ? slice?.label : undefined}
                      {...tap(slice)}
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
              <div key={s.key} className={groupsView ? 'legend-row legend-tap' : 'legend-row'} {...tap(s)}>
                <span className="dot" style={{ background: palette[i % palette.length] }} />
                <span className="legend-name">{s.label}</span>
                <Money minor={s.totalMinor} currency={currency} />
                <span className="legend-pct">{percent(s.totalMinor, total)}%</span>
                {groupsView && <Chevron />}
              </div>
            ))}
          </div>
        </div>
      )}
    </>
  )
}
