import { useCallback, useEffect, useMemo, useState } from 'react'
import { AccountsScreen } from './AccountsScreen'
import {
  fetchAccounts,
  fetchCategories,
  fetchMe,
  fetchTransactions,
  type Account,
  type Category,
  type Me,
  type Transaction,
} from './api'
import { CategoriesScreen } from './CategoriesScreen'
import { CurrencyScreen } from './CurrencyScreen'
import { EntryForm } from './EntryForm'
import { t } from './i18n'
import { dayIn, toEntries, type Entry } from './ledger'
import { PrivacyContext } from './MoneyText'
import { OperationsScreen } from './OperationsScreen'
import { TabBar, type Tab } from './TabBar'

type Data = {
  me: Me
  accounts: Account[]
  categories: Category[]
  transactions: Transaction[]
}

type State = { kind: 'loading' } | { kind: 'outside' } | { kind: 'error' } | { kind: 'ready'; data: Data }

type Screen = { kind: 'tabs' } | { kind: 'add' } | { kind: 'edit'; entry: Entry }

const HIDDEN_KEY = 'mk.hidden'

function readHidden(): boolean {
  try {
    return localStorage.getItem(HIDDEN_KEY) === '1'
  } catch {
    return false // хранилище может быть недоступно, режим тогда живёт до закрытия
  }
}

export function App() {
  const [state, setState] = useState<State>({ kind: 'loading' })
  const [tab, setTab] = useState<Tab>('operations')
  const [screen, setScreen] = useState<Screen>({ kind: 'tabs' })
  const [hidden, setHidden] = useState(readHidden)

  const privacy = useMemo(
    () => ({
      hidden,
      toggle: () =>
        setHidden((h) => {
          try {
            localStorage.setItem(HIDDEN_KEY, h ? '0' : '1')
          } catch {
            // не сохранилось, не страшно
          }
          return !h
        }),
    }),
    [hidden],
  )

  const load = useCallback(async () => {
    try {
      const me = await fetchMe()
      if (me.budget.base_currency === null) {
        setState({ kind: 'ready', data: { me, accounts: [], categories: [], transactions: [] } })
        return
      }
      const [accounts, categories, transactions] = await Promise.all([fetchAccounts(), fetchCategories(), fetchTransactions()])
      setState({ kind: 'ready', data: { me, accounts, categories, transactions } })
    } catch {
      setState({ kind: 'error' })
    }
  }, [])

  useEffect(() => {
    const webApp = window.Telegram?.WebApp
    if (webApp === undefined || webApp.initData === '') {
      setState({ kind: 'outside' })
      return
    }
    webApp.ready()
    void load()
  }, [load])

  const entries = useMemo(() => (state.kind === 'ready' ? toEntries(state.data.transactions) : []), [state])

  switch (state.kind) {
    case 'loading':
      return <p className="screen hint">{t.loading}</p>
    case 'outside':
      return <p className="screen hint">{t.openInTelegram}</p>
    case 'error':
      return (
        <main className="screen">
          <p className="error">{t.errors.internal}</p>
          <button type="button" className="primary" onClick={() => void load()}>
            {t.retry}
          </button>
        </main>
      )
    case 'ready': {
      const { me, accounts, categories } = state.data
      const base = me.budget.base_currency
      if (base === null) {
        return <CurrencyScreen currencies={me.base_currencies} onDone={() => void load()} />
      }

      const back = () => setScreen({ kind: 'tabs' })
      const done = () => {
        back()
        void load()
      }

      // Пока счетов нет, записывать некуда: «+» ведёт на вкладку счетов, где форма уже открыта.
      const onAdd = () => {
        if (accounts.length === 0) setTab('accounts')
        else setScreen({ kind: 'add' })
      }

      return (
        <PrivacyContext.Provider value={privacy}>
          {screen.kind === 'add' && <EntryForm accounts={accounts} categories={categories} onDone={done} onCancel={back} />}
          {screen.kind === 'edit' && <EntryForm accounts={accounts} categories={categories} entry={screen.entry} onDone={done} onCancel={back} />}
          {screen.kind === 'tabs' && (
            <>
              {tab === 'operations' && (
                <OperationsScreen
                  entries={entries}
                  accounts={accounts}
                  categories={categories}
                  today={dayIn(me.timezone)}
                  onOpen={(entry) => setScreen({ kind: 'edit', entry })}
                />
              )}
              {tab === 'categories' && <CategoriesScreen categories={categories} onChanged={() => void load()} />}
              {tab === 'accounts' && <AccountsScreen accounts={accounts} currencies={me.currencies} baseCurrency={base} onChanged={() => void load()} />}
              <TabBar tab={tab} onTab={setTab} onAdd={onAdd} />
            </>
          )}
        </PrivacyContext.Provider>
      )
    }
  }
}
