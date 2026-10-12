import type { Account, Category } from './api'
import { EntryList } from './EntryList'
import { t } from './i18n'
import type { Entry } from './ledger'

type Props = {
  entries: Entry[]
  accounts: Account[]
  categories: Category[]
  today: string
  onOpen: (entry: Entry) => void
}

export function OperationsScreen({ entries, accounts, categories, today, onOpen }: Props) {
  return (
    <main className="screen">
      <header className="row">
        <h1>{t.ledger.title}</h1>
      </header>

      {entries.length === 0 && <p className="hint">{t.ledger.empty}</p>}
      <EntryList entries={entries} accounts={accounts} categories={categories} today={today} onOpen={onOpen} />
    </main>
  )
}
