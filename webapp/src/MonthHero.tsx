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
  // Сколько дней подряд до сегодня есть записи (см. recordStreak).
  streak: number
  currency: string
  // Сколько категорий показать в списке под плитками.
  top?: number
}

const spentOf = (m: BudgetMonth) => m.categories.reduce((sum, r) => sum + Math.max(r.spent_minor, 0), 0)

// Линия нарастающих трат: плавная ломаная на всю ширину, последняя точка отмечена.
function Line({ values }: { values: number[] }) {
  const max = Math.max(...values, 0)
  if (values.length < 2 || max === 0) return null
  const w = 300
  const h = 36
  const pts = values.map((v, i) => [(i / (values.length - 1)) * w, h - 4 - (v / max) * (h - 10)] as const)
  const last = pts[pts.length - 1] ?? [0, 0]
  return (
    <svg className="bt-line" viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none" role="img" aria-label={t.hero.line}>
      <polyline points={pts.map((p) => p.join(',')).join(' ')} fill="none" stroke="#fff" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" vectorEffect="non-scaling-stroke" />
      <circle cx={last[0]} cy={last[1]} r="3.5" fill="#fff" vectorEffect="non-scaling-stroke" />
    </svg>
  )
}

const RING_R = 38
const RING_C = 2 * Math.PI * RING_R

// Кольцо долей крупнейших категорий; остальное серым куском.
function Ring({ parts }: { parts: { key: number; share: number; color: string }[] }) {
  let offset = 0
  return (
    <svg className="bt-donut" viewBox="0 0 104 104" role="img" aria-label={t.hero.byCategory}>
      <circle cx="52" cy="52" r={RING_R} fill="none" stroke="var(--line)" strokeWidth="14" />
      {parts.map((p) => {
        const len = p.share * RING_C
        const el = (
          <circle key={p.key} cx="52" cy="52" r={RING_R} fill="none" stroke={p.color} strokeWidth="14" strokeDasharray={`${Math.max(len - 3, 1)} ${RING_C}`} strokeDashoffset={-offset} transform="rotate(-90 52 52)" />
        )
        offset += len
        return el
      })}
    </svg>
  )
}

// MonthHero: главный экран «Операций» плитками «Бенто»: потрачено за месяц с линией, кольцо категорий, средний день,
// серия дней с записями и три крупнейшие категории.
export function MonthHero({ month, previous, categories, line, streak, currency, top = 3 }: Props) {
  const colors = useContext(CategoryColorContext)
  const spent = month === null ? 0 : spentOf(month)
  const before = previous === null ? null : spentOf(previous)
  const delta = before === null ? 0 : spent - before
  const days = Math.max(line.length, 1)
  const name = new Map(categories.map((c) => [c.id, c.name]))
  const sorted = (month?.categories ?? []).filter((r) => r.spent_minor > 0 && name.has(r.category_id)).sort((a, b) => b.spent_minor - a.spent_minor)
  const biggest = sorted.slice(0, top)
  const ringParts = sorted.slice(0, 4).map((r) => ({ key: r.category_id, share: spent === 0 ? 0 : r.spent_minor / spent, color: colors.get(r.category_id) ?? 'var(--accent)' }))

  return (
    <div className="bento">
      <div className="bt bt-hero">
        <div className="bt-top">
          <span>{t.hero.spent}</span>
          {before !== null && before > 0 && delta !== 0 && (
            <span className="bt-pill">
              {delta < 0 ? '−' : '+'}
              <Money minor={Math.abs(delta)} currency={currency} /> {t.hero.toPrevious}
            </span>
          )}
        </div>
        {month === null ? <Skeleton w={160} h={40} /> : <Money minor={spent} currency={currency} />}
        <Line values={line} />
      </div>

      <div className="bt bt-ring">
        <span className="bt-cap">{t.hero.byCategory}</span>
        {ringParts.length > 0 ? <Ring parts={ringParts} /> : <p className="hint tight">{t.reports.empty}</p>}
      </div>

      <div className="bt-col">
        <div className="bt bt-lime">
          <span className="bt-cap">{t.hero.avgDay}</span>
          {month === null ? <Skeleton w={80} h={26} /> : <Money minor={Math.round(spent / days)} currency={currency} />}
        </div>
        <div className="bt bt-peach">
          <span className="bt-cap">{t.hero.streak}</span>
          <b>{streak}</b>
        </div>
      </div>

      {biggest.length > 0 && (
        <div className="card bt-list">
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
      )}
    </div>
  )
}
