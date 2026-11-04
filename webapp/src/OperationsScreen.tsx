import { useEffect, useState, type ReactNode } from 'react'
import { fetchBudgetMonth, type Account, type BudgetMonth, type Category } from './api'
import { totalBalance } from './accounts'
import { EntryList } from './EntryList'
import { t } from './i18n'
import { dayNet, entryMatches, lastDays, type Entry } from './ledger'
import { Money } from './MoneyText'
import { EmptyState, Skeleton } from './States'

type Props = {
  entries: Entry[]
  accounts: Account[]
  categories: Category[]
  today: string
  // Валюта пересчёта остатков и записей, и валюта бюджета для итогов месяца.
  display: string
  base: string
  onOpen: (entry: Entry) => void
  onAdd: () => void
  onOpenDay?: (date: string) => void
  onOpenMore: () => void
  wrapRow?: (entry: Entry, row: ReactNode) => ReactNode
}

function weekday(date: string): string {
  const [y, m, d] = date.split('-').map(Number)
  return new Date(y ?? 1970, (m ?? 1) - 1, d ?? 1).toLocaleDateString('ru-RU', { weekday: 'short' })
}

export function OperationsScreen({ entries, accounts, categories, today, display, base, onOpen, onAdd, onOpenDay, onOpenMore, wrapRow }: Props) {
  const [month, setMonth] = useState<BudgetMonth | null>(null)
  const [searching, setSearching] = useState(false)
  const [query, setQuery] = useState('')
  const [accountId, setAccountId] = useState<number | null>(null)

  // Итоги месяца берутся с сервера целиком: в списке записей только последние.
  useEffect(() => {
    let alive = true
    fetchBudgetMonth(today.slice(0, 7))
      .then((b) => alive && setMonth(b))
      .catch(() => {})
    return () => {
      alive = false
    }
  }, [today, entries])

  const accountName = new Map(accounts.map((a) => [a.id, a.name]))
  const categoryName = new Map(categories.map((c) => [c.id, c.name]))
  const balance = totalBalance(accounts)
  const spent = month === null ? 0 : month.categories.reduce((sum, r) => sum + Math.max(r.spent_minor, 0), 0)
  const week = lastDays(entries, today, 7)
  const weekMax = Math.max(...week.map((d) => d.spentMinor), 0)

  const filtering = query.trim() !== '' || accountId !== null
  const shown = entries.filter((e) => (accountId === null || e.accountId === accountId || e.toAccountId === accountId) && entryMatches(e, query, accountName, categoryName))
  const net = filtering ? dayNet(shown) : null

  function closeSearch() {
    setSearching(false)
    setQuery('')
    setAccountId(null)
  }

  return (
    <main className="screen">
      <header className="row">
        <h1>{t.ledger.title}</h1>
        <span className="row-tight">
          <button type="button" className={searching ? 'plus plus-solid' : 'plus'} aria-label={t.ledger.search} aria-pressed={searching} onClick={() => (searching ? closeSearch() : setSearching(true))}>
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="M11 4a7 7 0 100 14 7 7 0 000-14zM16 16l4 4" />
            </svg>
          </button>
          <button type="button" className="plus" aria-label={t.more.title} onClick={onOpenMore}>
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="M5 12h.01M12 12h.01M19 12h.01" />
            </svg>
          </button>
        </span>
      </header>

      {searching && (
        <>
          <label className="search-field">
            <input autoFocus type="search" inputMode="search" value={query} placeholder={t.ledger.searchPlaceholder} onChange={(e) => setQuery(e.target.value)} />
          </label>
          <div className="chips-row">
            <button type="button" className={accountId === null ? 'chip chip-on' : 'chip'} onClick={() => setAccountId(null)}>
              {t.ledger.allAccounts}
            </button>
            {accounts
              .filter((a) => !a.archived)
              .map((a) => (
                <button key={a.id} type="button" className={accountId === a.id ? 'chip chip-on' : 'chip'} onClick={() => setAccountId(accountId === a.id ? null : a.id)}>
                  {a.name}
                </button>
              ))}
          </div>
        </>
      )}

      {!filtering && entries.length > 0 && (
        <div className="card ops-hero">
          <span className="label">{t.ledger.balance}</span>
          {balance === null ? <Skeleton w={160} h={34} /> : <Money minor={balance} currency={display} />}
          <div className="duo">
            <div>
              <span className="label">{t.ledger.spentMonth}</span>
              {month === null ? <Skeleton w={90} h={18} /> : <Money minor={-spent} currency={base} />}
            </div>
            <div>
              <span className="label">{t.ledger.incomeMonth}</span>
              {month === null ? <Skeleton w={90} h={18} /> : <Money minor={month.income_minor} currency={base} signed />}
            </div>
          </div>
          {weekMax > 0 && (
            <>
              <span className="label week-label">{t.ledger.week}</span>
              <div className="week" role="img" aria-label={t.ledger.week}>
                {week.map((d) => (
                  <div key={d.date} className={d.date === today ? 'today' : ''}>
                    <i style={{ height: Math.max(4, Math.round((d.spentMinor / weekMax) * 44)) }} />
                    <small>{weekday(d.date)}</small>
                  </div>
                ))}
              </div>
            </>
          )}
        </div>
      )}

      {filtering && (
        <p className="found">
          {t.ledger.found}: <b>{shown.length}</b>
          {net !== null && (
            <>
              {' · '}
              <Money minor={net} currency={display} signed />
            </>
          )}
        </p>
      )}

      {entries.length === 0 && <EmptyState title={t.ledger.emptyTitle} text={t.ledger.emptyText} action={{ label: t.ledger.addFirst, onClick: onAdd }} />}
      {entries.length > 0 && shown.length === 0 && <EmptyState title={t.ledger.nothing} text={t.ledger.nothingHint} />}
      <EntryList entries={shown} accounts={accounts} categories={categories} today={today} onOpen={onOpen} totalCurrency={display} onOpenDay={onOpenDay} wrapRow={wrapRow} />
    </main>
  )
}
