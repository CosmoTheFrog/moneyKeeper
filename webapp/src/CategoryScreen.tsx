import type { Account, Category } from './api'
import { EntryList } from './EntryList'
import { t } from './i18n'
import type { Entry } from './ledger'

type Props = {
  category: Category
  entries: Entry[]
  accounts: Account[]
  categories: Category[]
  today: string
  onBack: () => void
  onAdd: () => void
  onOpen: (entry: Entry) => void
}

// Траты по одной категории, новые сверху. «+» добавляет трату сразу в эту категорию.
export function CategoryScreen({ category, entries, accounts, categories, today, onBack, onAdd, onOpen }: Props) {
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
      <p className="hint">
        {t.groups[category.group]} · {t.categories.spendHere}
      </p>

      {entries.length === 0 && <p className="hint">{t.categories.noSpend}</p>}
      <EntryList entries={entries} accounts={accounts} categories={categories} today={today} onOpen={onOpen} />
    </main>
  )
}
