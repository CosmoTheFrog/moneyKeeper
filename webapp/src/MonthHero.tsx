import { useContext, type CSSProperties } from 'react'
import type { BudgetMonth, Category } from './api'
import { CategoryColorContext } from './categoryColor'
import { categoryIcon } from './categoryIcons'
import { t } from './i18n'
import { Money } from './MoneyText'
import { Skeleton } from './States'

type Props = {
  month: BudgetMonth | null
  previous: BudgetMonth | null
  categories: Category[]
  // Траты месяца нарастающим итогом по дням (см. monthLine).
  line: number[]
  currency: string
  // Сколько категорий показать под линией.
  top?: number
}

const spentOf = (m: BudgetMonth) => m.categories.reduce((sum, r) => sum + Math.max(r.spent_minor, 0), 0)

// Линия нарастающих трат: плавная ломаная на всю ширину, последняя точка отмечена.
function Line({ values }: { values: number[] }) {
  const max = Math.max(...values, 0)
  if (values.length < 2 || max === 0) return null
  const w = 300
  const h = 56
  const pts = values.map((v, i) => [(i / (values.length - 1)) * w, h - 4 - (v / max) * (h - 12)] as const)
  const last = pts[pts.length - 1] ?? [0, 0]
  return (
    <svg className="hero-line" viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none" role="img" aria-label={t.hero.line}>
      <polyline points={pts.map((p) => p.join(',')).join(' ')} fill="none" stroke="var(--accent)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" vectorEffect="non-scaling-stroke" />
      <circle cx={last[0]} cy={last[1]} r="3" fill="var(--accent)" vectorEffect="non-scaling-stroke" />
    </svg>
  )
}

// MonthHero: главный блок «Операций»: потрачено за месяц, сравнение с прошлым, линия по дням и три категории с самыми большими тратами.
export function MonthHero({ month, previous, categories, line, currency, top = 3 }: Props) {
  const colors = useContext(CategoryColorContext)
  const spent = month === null ? 0 : spentOf(month)
  const before = previous === null ? null : spentOf(previous)
  const delta = before === null ? 0 : spent - before
  const name = new Map(categories.map((c) => [c.id, c.name]))
  const biggest = (month?.categories ?? [])
    .filter((r) => r.spent_minor > 0 && name.has(r.category_id))
    .sort((a, b) => b.spent_minor - a.spent_minor)
    .slice(0, top)

  return (
    <div className="card hero">
      <span className="label">{t.ledger.spentMonth}</span>
      {month === null ? <Skeleton w={160} h={34} /> : <Money minor={-spent} currency={currency} />}
      {before !== null && before > 0 && delta !== 0 && (
        <p className={delta < 0 ? 'hero-delta hero-good' : 'hero-delta'}>
          {t.budget.vsPrevious.by} <Money minor={Math.abs(delta)} currency={currency} /> {delta > 0 ? t.budget.vsPrevious.more : t.budget.vsPrevious.less}
        </p>
      )}
      <Line values={line} />
      {biggest.map((r) => {
        const title = name.get(r.category_id) ?? ''
        const path = categoryIcon(title)
        return (
          <div key={r.category_id} className="hero-row">
            <span className="avatar hero-av" style={{ '--gc': colors.get(r.category_id) ?? 'var(--accent)' } as CSSProperties} aria-hidden="true">
              {path !== null ? (
                <svg viewBox="0 0 24 24">
                  <path d={path} />
                </svg>
              ) : (
                <b>{[...title.trim()][0]?.toUpperCase() ?? ''}</b>
              )}
            </span>
            <span className="hero-name">{title}</span>
            <Money minor={-r.spent_minor} currency={currency} />
          </div>
        )
      })}
    </div>
  )
}
