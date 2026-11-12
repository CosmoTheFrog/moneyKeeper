import { useEffect, useState, type ReactNode } from 'react'
import { fetchBudgetMonth, type Account, type BudgetMonth, type Category } from './api'
import { totalBalance } from './accounts'
import { shiftMonth } from './calendar'
import { EntryList } from './EntryList'
import { t } from './i18n'
import { dayNet, entryMatches, type Entry } from './ledger'
import { MonthHero } from './MonthHero'
import { Money } from './MoneyText'
import { monthLine } from './report'
import { EmptyState } from './States'

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
  // Простой режим: остатка по счетам нет, показывать его не нужно.
  simple?: boolean
  wrapRow?: (entry: Entry, row: ReactNode) => ReactNode
}

export function OperationsScreen({ entries, accounts, categories, today, display, base, onOpen, onAdd, onOpenDay, onOpenMore, simple = false, wrapRow }: Props) {
  const [month, setMonth] = useState<BudgetMonth | null>(null)
  const [previous, setPrevious] = useState<BudgetMonth | null>(null)
  const [searching, setSearching] = useState(false)
  const [query, setQuery] = useState('')
  const [accountId, setAccountId] = useState<number | null>(null)

  // Итоги месяца берутся с сервера целиком: в списке записей только последние.
  useEffect(() => {
    let alive = true
    fetchBudgetMonth(today.slice(0, 7))
      .then((b) => alive && setMonth(b))
      .catch(() => {})
    fetchBudgetMonth(shiftMonth(today.slice(0, 7), -1))
      .then((b) => alive && setPrevious(b))
      .catch(() => {})
    return () => {
      alive = false
    }
  }, [today, entries])

  const accountName = new Map(accounts.map((a) => [a.id, a.name]))
  const categoryName = new Map(categories.map((c) => [c.id, c.name]))
  const balance = totalBalance(accounts)

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
        <>
          <MonthHero month={month} previous={previous} categories={categories} line={monthLine(entries, today)} currency={base} />
          {!simple && balance !== null && (
            <p className="hero-balance">
              {t.ledger.balance} <Money minor={balance} currency={display} />
            </p>
          )}
        </>
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
