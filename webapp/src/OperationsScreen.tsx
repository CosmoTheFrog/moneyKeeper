import { useContext } from 'react'
import type { Account, Category } from './api'
import type { Entry } from './ledger'
import { groupByDay } from './ledger'
import { t } from './i18n'
import { Money, PrivacyContext } from './MoneyText'

type Props = {
  entries: Entry[]
  accounts: Account[]
  categories: Category[]
  today: string
  onOpen: (entry: Entry) => void
}

function dayLabel(date: string, today: string): string {
  if (date === today) return t.today
  const [y, m, d] = date.split('-').map(Number)
  return new Date(y ?? 1970, (m ?? 1) - 1, d ?? 1).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' })
}

export function OperationsScreen({ entries, accounts, categories, today, onOpen }: Props) {
  const { hidden, toggle } = useContext(PrivacyContext)
  const accountName = new Map(accounts.map((a) => [a.id, a.name]))
  const categoryName = new Map(categories.map((c) => [c.id, c.name]))

  function title(e: Entry): string {
    switch (e.kind) {
      case 'transfer':
        return `${accountName.get(e.accountId) ?? ''} → ${e.toAccountId === null ? '…' : (accountName.get(e.toAccountId) ?? '')}`
      case 'opening':
        return t.ledger.opening
      case 'income':
        return t.ledger.income
      case 'expense':
        return (e.categoryId !== null ? categoryName.get(e.categoryId) : undefined) ?? t.ledger.uncategorized
    }
  }

  function subtitle(e: Entry): string {
    const account = e.kind === 'transfer' ? '' : (accountName.get(e.accountId) ?? '')
    return [account, e.note].filter((s) => s !== '').join(' · ')
  }

  return (
    <main className="screen">
      <header className="row">
        <h1>{t.ledger.title}</h1>
        <button type="button" className="chip" onClick={toggle}>
          {hidden ? t.privacy.show : t.privacy.hide}
        </button>
      </header>

      {entries.length === 0 && <p className="hint">{t.ledger.empty}</p>}

      {groupByDay(entries).map((g) => (
        <section key={g.date}>
          <h2 className="day">{dayLabel(g.date, today)}</h2>
          <div className="card list-card">
            {g.entries.map((e) => (
              <button key={e.id} type="button" className="entry" onClick={() => onOpen(e)}>
                <span className="entry-main">
                  <span className="entry-title">{title(e)}</span>
                  <span className="entry-sub">{subtitle(e)}</span>
                </span>
                <Money minor={e.amountMinor} currency={e.currency} signed={e.kind !== 'transfer'} />
              </button>
            ))}
          </div>
        </section>
      ))}
    </main>
  )
}
