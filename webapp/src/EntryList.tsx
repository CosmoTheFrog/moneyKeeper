import type { Account, Category } from './api'
import { groupByDay, type Entry } from './ledger'
import { t } from './i18n'
import { Money } from './MoneyText'

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

// Список записей по дням. Нажатие на строку открывает запись, нажатие на сумму скрывает или показывает все суммы.
export function EntryList({ entries, accounts, categories, today, onOpen }: Props) {
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
    <>
      {groupByDay(entries).map((g) => (
        <section key={g.date}>
          <h2 className="day">{dayLabel(g.date, today)}</h2>
          <div className="card list-card">
            {g.entries.map((e) => (
              <div key={e.id} className="entry">
                <button type="button" className="entry-open" onClick={() => onOpen(e)}>
                  <span className="entry-title">{title(e)}</span>
                  <span className="entry-sub">{subtitle(e)}</span>
                </button>
                <Money minor={e.amountMinor} currency={e.currency} signed={e.kind !== 'transfer'} />
              </div>
            ))}
          </div>
        </section>
      ))}
    </>
  )
}
