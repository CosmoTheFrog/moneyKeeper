import { useContext, type CSSProperties } from 'react'
import type { Account, Category } from './api'
import { EntryAvatar } from './EntryAvatar'
import { EntryList } from './EntryList'
import { CategoryColorContext } from './groupColor'
import { t } from './i18n'
import { dayParts, daySummary, shiftDay, type Entry } from './ledger'
import { Chevron } from './MonthNav'
import { Money } from './MoneyText'

type Props = {
  date: string
  entries: Entry[]
  accounts: Account[]
  categories: Category[]
  today: string
  // Валюта пересчёта: в ней суммы дня.
  display: string
  onBack: () => void
  onShift: (date: string) => void
  onOpen: (entry: Entry) => void
  onAdd: (date: string) => void
}

function parts(date: string): Date {
  const [y, m, d] = date.split('-').map(Number)
  return new Date(y ?? 1970, (m ?? 1) - 1, d ?? 1)
}

// Один день: сколько потрачено, куда ушло и все записи дня.
export function DayScreen({ date, entries, accounts, categories, today, display, onBack, onShift, onOpen, onAdd }: Props) {
  const colors = useContext(CategoryColorContext)
  const day = entries.filter((e) => e.date === date)
  const sum = daySummary(day)
  const split = dayParts(day)
  const splitTotal = split.reduce((s, p) => s + p.totalMinor, 0)
  const categoryName = new Map(categories.map((c) => [c.id, c.name]))
  const when = parts(date)
  const sub = date === today ? t.today : when.toLocaleDateString('ru-RU', { weekday: 'long' })

  return (
    <main className="screen">
      <header className="row">
        <button type="button" className="chip" onClick={onBack}>
          {t.back}
        </button>
      </header>
      <h1>{when.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' })}</h1>

      <div className="month-nav">
        <button type="button" className="nav-arrow" aria-label={t.day.prev} onClick={() => onShift(shiftDay(date, -1))}>
          <Chevron left />
        </button>
        <span className="month-title">{sub}</span>
        <button type="button" className="nav-arrow" aria-label={t.day.next} disabled={date >= today} onClick={() => onShift(shiftDay(date, 1))}>
          <Chevron left={false} />
        </button>
      </div>

      <div className="card ops-hero">
        <span className="label">{t.day.spent}</span>
        <Money minor={-sum.spentMinor} currency={display} />
        <div className="duo">
          <div>
            <span className="label">{t.day.income}</span>
            <Money minor={sum.incomeMinor} currency={display} signed />
          </div>
          <div>
            <span className="label">{t.day.records}</span>
            <b className="duo-count">{day.length}</b>
          </div>
        </div>
        {sum.missingRates > 0 && (
          <p className="hint tight">
            {t.reports.missingRates}: {sum.missingRates}
          </p>
        )}
      </div>

      {split.length > 0 && (
        <>
          <h2 className="day">{t.day.where}</h2>
          <div className="card">
            <div className="split" aria-hidden="true">
              {split.map((p) => (
                <span key={p.categoryId ?? 'none'} className="split-part" style={{ width: `${(p.totalMinor / splitTotal) * 100}%`, background: p.categoryId !== null ? (colors.get(p.categoryId) ?? 'var(--accent)') : 'var(--muted)' }} />
              ))}
            </div>
            <div className="part-list">
              {split.map((p) => {
                const name = p.categoryId !== null ? (categoryName.get(p.categoryId) ?? t.ledger.uncategorized) : t.ledger.uncategorized
                const stub: Entry = { id: 0, kind: 'expense', date, amountMinor: 0, currency: display, baseMinor: 0, accountId: 0, toAccountId: null, categoryId: p.categoryId, note: '', inMinor: null, inCurrency: null, feeMinor: 0 }
                return (
                  <div key={p.categoryId ?? 'none'} className="part-row">
                    <EntryAvatar entry={stub} categoryName={name} color={p.categoryId !== null ? colors.get(p.categoryId) : undefined} />
                    <span className="part-name">{name}</span>
                    <small>{Math.round((p.totalMinor / splitTotal) * 100)}%</small>
                    <Money minor={-p.totalMinor} currency={display} />
                  </div>
                )
              })}
            </div>
          </div>
        </>
      )}

      <h2 className="day">
        {t.day.records} · {day.length}
      </h2>
      {day.length === 0 && <p className="hint">{t.day.empty}</p>}
      <EntryList entries={day} accounts={accounts} categories={categories} today={today} onOpen={onOpen} bare />

      <button type="button" className="primary day-cta" style={{ '--cta-bottom': '24px' } as CSSProperties} onClick={() => onAdd(date)}>
        + {t.day.add}
      </button>
    </main>
  )
}
