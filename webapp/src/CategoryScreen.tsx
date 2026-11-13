import type { Account, Category } from './api'
import { EntryList } from './EntryList'
import { t } from './i18n'
import { categoryTotals, type Entry } from './ledger'
import { Money } from './MoneyText'

type Props = {
  category: Category
  entries: Entry[]
  accounts: Account[]
  categories: Category[]
  today: string
  // Валюта, в которой посчитаны суммы записей (baseMinor).
  currency: string
  onBack: () => void
  onAdd: () => void
  onOpen: (entry: Entry) => void
}

// Траты по одной категории, новые сверху. «+» добавляет трату сразу в эту категорию.
export function CategoryScreen({ category, entries, accounts, categories, today, currency, onBack, onAdd, onOpen }: Props) {
  const totals = categoryTotals(entries, today)
  return (
    <main className="screen">
      <header className="row">
        <button type="button" className="chip" onClick={onBack}>
          {t.back}
        </button>
        <button type="button" className="plus plus-solid" aria-label={t.categories.addExpense} onClick={onAdd}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.6" strokeLinecap="round" aria-hidden="true">
            <path d="M12 5v14M5 12h14" />
          </svg>
        </button>
      </header>

      <h1>{category.name}</h1>
      <p className="hint">{t.categories.spendHere}</p>
      <div className="bento">
        <div className="bt bt-lime">
          <span className="bt-cap">{t.categories.thisMonth}</span>
          <Money minor={totals.month} currency={currency} />
        </div>
        <div className="bt bt-peach">
          <span className="bt-cap">{t.categories.allTime}</span>
          <Money minor={totals.all} currency={currency} />
        </div>
      </div>

      {entries.length === 0 && <p className="hint">{t.categories.noSpend}</p>}
      <EntryList entries={entries} accounts={accounts} categories={categories} today={today} onOpen={onOpen} noteFirst="plain" />
    </main>
  )
}
