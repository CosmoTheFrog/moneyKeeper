import { useEffect, useState, type ReactNode } from 'react'
import { fetchBudgetMonth, fetchOccurrences, type Account, type BudgetMonth, type Category, type Occurrence } from './api'
import { totalBalance } from './accounts'
import { DisplayCurrency } from './DisplayCurrency'
import { EntryList } from './EntryList'
import { FullTiles } from './FullTiles'
import { t } from './i18n'
import { dayNet, entryMatches, recordStreak, type Entry } from './ledger'
import { MonthHero } from './MonthHero'
import { Money } from './MoneyText'
import { scaleMonth, useDisplayFactor } from './displayFactor'
import { onKey } from './onKey'
import { monthLine, previousRange, spendByCategory } from './report'
import { EmptyState } from './States'
import type { Tab } from './TabBar'

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
  // Выбор валюты показа рядом с поиском.
  currencies: string[]
  onDisplay: (code: string) => void
  // Плитки главного экрана ведут в другие разделы.
  onOpenTab: (tab: Tab) => void
  onOpenCategory: (id: number) => void
  onOpenSchedules: () => void
  onOpenSavings: () => void
  // Имя в профиле: первая буква стоит на кнопке «Ещё».
  name: string
  // Простой режим: остатка по счетам нет, показывать его не нужно.
  simple?: boolean
  wrapRow?: (entry: Entry, row: ReactNode) => ReactNode
}

export function OperationsScreen({ entries, accounts, categories, today, display, base, onOpen, onAdd, onOpenDay, onOpenMore, currencies, onDisplay, onOpenTab, onOpenCategory, onOpenSchedules, onOpenSavings, name, simple = false, wrapRow }: Props) {
  const [month, setMonth] = useState<BudgetMonth | null>(null)
  const [nextPayment, setNextPayment] = useState<Occurrence | null>(null)
  const [searching, setSearching] = useState(false)
  const [query, setQuery] = useState('')
  const [accountId, setAccountId] = useState<number | null>(null)

  // Итоги месяца берутся с сервера целиком: в списке записей только последние.
  useEffect(() => {
    let alive = true
    fetchBudgetMonth(today.slice(0, 7))
      .then((b) => alive && setMonth(b))
      .catch(() => {})
    if (!simple) {
      fetchOccurrences()
        .then((list) => alive && setNextPayment(list.filter((o) => o.status === 'upcoming' || o.status === 'overdue').sort((a, b) => a.due_on.localeCompare(b.due_on))[0] ?? null))
        .catch(() => {})
    }
    return () => {
      alive = false
    }
  }, [today, entries, simple])

  // Итоги месяца считаются в базовой валюте; в другой валюте показа пересчитываются по курсу дня.
  const factor = useDisplayFactor(display, base)
  const accountName = new Map(accounts.map((a) => [a.id, a.name]))
  const categoryName = new Map(categories.map((c) => [c.id, c.name]))
  const balance = totalBalance(accounts)
  // Прошлый месяц до сегодняшнего числа, чтобы сравнение с неполным текущим месяцем было честным.
  const previousMinor = spendByCategory(entries, categories, previousRange('month', today), { kind: 'base' }).reduce((s, x) => s + x.totalMinor, 0)

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
          <DisplayCurrency display={display} currencies={currencies} onChange={onDisplay} />
          <button type="button" className={searching ? 'plus plus-solid' : 'plus'} aria-label={t.ledger.search} aria-pressed={searching} onClick={() => (searching ? closeSearch() : setSearching(true))}>
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="M11 4a7 7 0 100 14 7 7 0 000-14zM16 16l4 4" />
            </svg>
          </button>
          <button type="button" className="plus plus-avatar" aria-label={t.more.title} onClick={onOpenMore}>
            {[...name.trim()][0]?.toUpperCase() ?? ''}
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
          <MonthHero month={scaleMonth(month, factor)} previousMinor={previousMinor} categories={categories} line={monthLine(entries, today)} streak={recordStreak(entries, today)} currency={display} onOpenReports={() => onOpenTab('reports')} onOpenBudget={() => onOpenTab('budget')} onOpenToday={() => onOpenDay?.(today)} onAdd={onAdd} onOpenCategory={onOpenCategory} />
          {!simple && <FullTiles accounts={accounts.filter((a) => !a.archived)} nextPayment={nextPayment} today={today} onOpenAccounts={() => onOpenTab('accounts')} onOpenSchedules={onOpenSchedules} onOpenSavings={onOpenSavings} />}
          {!simple && balance !== null && (
            <div role="button" tabIndex={0} className="hero-balance" onClick={() => onOpenTab('accounts')} onKeyDown={onKey(() => onOpenTab('accounts'))}>
              {t.ledger.balance} <Money minor={balance} currency={display} />
            </div>
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
