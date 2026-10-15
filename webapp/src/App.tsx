import { useCallback, useEffect, useMemo, useState } from 'react'
import { AccountReport } from './AccountReport'
import { AccountsScreen } from './AccountsScreen'
import {
  fetchAccounts,
  fetchCategories,
  fetchCategoryGroups,
  fetchMe,
  fetchTransactions,
  type Account,
  type Category,
  type CategoryGroup,
  type Me,
  type Transaction,
} from './api'
import { BudgetScreen } from './BudgetScreen'
import { CategoryScreen } from './CategoryScreen'
import { CurrencyScreen } from './CurrencyScreen'
import { EntryForm } from './EntryForm'
import { t } from './i18n'
import { dayIn, toEntries, type Entry } from './ledger'
import { PrivacyContext } from './MoneyText'
import { registerCurrencies } from './money'
import { OperationsScreen } from './OperationsScreen'
import { PortfolioScreen } from './PortfolioScreen'
import { ReportsScreen } from './ReportsScreen'
import { SavingsScreen } from './SavingsScreen'
import { TabBar, type Tab } from './TabBar'
import { applyTheme, readMode, resolveTheme, saveMode, type ThemeMode } from './theme'
import { ThemeContext } from './ThemeSettings'

type Data = {
  me: Me
  accounts: Account[]
  groups: CategoryGroup[]
  categories: Category[]
  transactions: Transaction[]
}

type State = { kind: 'loading' } | { kind: 'outside' } | { kind: 'error' } | { kind: 'ready'; data: Data }

type Screen =
  | { kind: 'tabs' }
  | { kind: 'add'; categoryId?: number; returnTo: Screen }
  | { kind: 'edit'; entry: Entry; returnTo: Screen }
  | { kind: 'category'; categoryId: number }
  | { kind: 'account'; accountId: number }
  | { kind: 'savings' }
  | { kind: 'portfolio' }

const HIDDEN_KEY = 'mk.hidden'
const DISPLAY_KEY = 'mk.display'

// Валюта, в которой показаны отчёты и итоги. Пусто значит базовая валюта бюджета.
function readDisplay(): string {
  try {
    return localStorage.getItem(DISPLAY_KEY) ?? ''
  } catch {
    return ''
  }
}

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
  const [themeMode, setThemeMode] = useState<ThemeMode>(readMode)
  const [displayChoice, setDisplayChoice] = useState(readDisplay)

  const theme = useMemo(
    () => ({
      mode: themeMode,
      setMode: (mode: ThemeMode) => {
        saveMode(mode)
        setThemeMode(mode)
      },
    }),
    [themeMode],
  )

  // Тема применяется к странице целиком; «как в Telegram» берёт схему мессенджера, без него системную.
  useEffect(() => {
    const dark = window.matchMedia('(prefers-color-scheme: dark)')
    const apply = () => applyTheme(resolveTheme(themeMode, window.Telegram?.WebApp.colorScheme, dark.matches))
    apply()
    dark.addEventListener('change', apply)
    return () => dark.removeEventListener('change', apply)
  }, [themeMode])

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
      registerCurrencies(me.currency_info)
      if (me.budget.base_currency === null) {
        setState({ kind: 'ready', data: { me, accounts: [], groups: [], categories: [], transactions: [] } })
        return
      }
      // Сохранённая валюта могла пропасть из списка: тогда считаем в базовой.
      const target = me.currencies.includes(displayChoice) ? displayChoice : ''
      const [accounts, groups, categories, transactions] = await Promise.all([fetchAccounts(target), fetchCategoryGroups(), fetchCategories(), fetchTransactions(target)])
      setState({ kind: 'ready', data: { me, accounts, groups, categories, transactions } })
    } catch {
      setState({ kind: 'error' })
    }
  }, [displayChoice])

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

  // Форма записи открывается окном поверх экрана, из которого её вызвали, этот экран остаётся под ней.
  const layer: Screen = screen.kind === 'add' || screen.kind === 'edit' ? screen.returnTo : screen

  // Новый экран открывается с самого верха, а окно поверх не сдвигает экран под собой.
  useEffect(() => {
    window.scrollTo(0, 0)
  }, [tab, layer.kind])

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
      const { me, accounts, groups, categories } = state.data
      const base = me.budget.base_currency
      const display = me.currencies.includes(displayChoice) ? displayChoice : (base ?? '')
      const changeDisplay = (code: string) => {
        try {
          localStorage.setItem(DISPLAY_KEY, code)
        } catch {
          // не сохранилось, не страшно
        }
        setDisplayChoice(code)
      }
      if (base === null) {
        return <CurrencyScreen currencies={me.base_currencies} onDone={() => void load()} />
      }

      const today = dayIn(me.timezone)
      const reload = () => void load()
      const go = (to: Screen) => setScreen(to)
      const home: Screen = { kind: 'tabs' }

      const openEntry = (entry: Entry, returnTo: Screen) => go({ kind: 'edit', entry, returnTo })
      const finish = (returnTo: Screen) => {
        go(returnTo)
        reload()
      }

      // Пока счетов нет, записывать некуда: «+» ведёт на вкладку счетов, где форма уже открыта.
      const onAdd = () => {
        if (accounts.length === 0) setTab('accounts')
        else go({ kind: 'add', returnTo: home })
      }

      let overlay = null
      if (screen.kind === 'add') {
        overlay = (
          <EntryForm
            accounts={accounts}
            groups={groups}
            categories={categories}
            allCurrencies={me.currencies}
            presetCategoryId={screen.categoryId}
            onDone={() => finish(screen.returnTo)}
            onCancel={() => go(screen.returnTo)}
          />
        )
      } else if (screen.kind === 'edit') {
        overlay = (
          <EntryForm
            accounts={accounts}
            groups={groups}
            categories={categories}
            allCurrencies={me.currencies}
            entry={screen.entry}
            onDone={() => finish(screen.returnTo)}
            onCancel={() => go(screen.returnTo)}
          />
        )
      }

      let body
      switch (layer.kind) {
        case 'category': {
          const category = categories.find((c) => c.id === layer.categoryId)
          if (category === undefined) {
            body = null
            break
          }
          const here: Screen = { kind: 'category', categoryId: category.id }
          body = (
            <CategoryScreen
              category={category}
              groupName={groups.find((g) => g.id === category.group_id)?.name ?? ''}
              entries={entries.filter((e) => e.categoryId === category.id)}
              accounts={accounts}
              categories={categories}
              today={today}
              onBack={() => go(home)}
              onAdd={() => go({ kind: 'add', categoryId: category.id, returnTo: here })}
              onOpen={(entry) => openEntry(entry, here)}
            />
          )
          break
        }
        case 'account': {
          const account = accounts.find((a) => a.id === layer.accountId)
          if (account === undefined) {
            body = null
            break
          }
          const here: Screen = { kind: 'account', accountId: account.id }
          body = (
            <AccountReport
              account={account}
              entries={entries}
              accounts={accounts}
              groups={groups}
              categories={categories}
              baseCurrency={display}
              currencies={me.currencies}
              onDisplay={changeDisplay}
              today={today}
              onBack={() => {
                setTab('accounts')
                go(home)
              }}
              onOpen={(entry) => openEntry(entry, here)}
            />
          )
          break
        }
        case 'savings':
          body = (
            <SavingsScreen
              accounts={accounts}
              allCurrencies={me.currencies}
              today={today}
              onBack={() => {
                setTab('accounts')
                go(home)
              }}
              onChanged={reload}
            />
          )
          break
        case 'portfolio':
          body = (
            <PortfolioScreen
              display={display}
              currencies={me.currencies}
              onDisplay={changeDisplay}
              onBack={() => {
                setTab('accounts')
                go(home)
              }}
            />
          )
          break
        case 'tabs':
          body = (
            <>
              {tab === 'operations' && <OperationsScreen entries={entries} accounts={accounts} categories={categories} today={today} onOpen={(entry) => openEntry(entry, home)} />}
              {tab === 'budget' && (
                <BudgetScreen
                  groups={groups}
                  categories={categories}
                  baseCurrency={base}
                  today={today}
                  ledger={state.data.transactions}
                  onChanged={reload}
                  onOpenCategory={(id) => go({ kind: 'category', categoryId: id })}
                  onAddTo={(id) => go({ kind: 'add', categoryId: id, returnTo: home })}
                />
              )}
              {tab === 'reports' && <ReportsScreen entries={entries} groups={groups} categories={categories} baseCurrency={display} currencies={me.currencies} onDisplay={changeDisplay} today={today} />}
              {tab === 'accounts' && <AccountsScreen accounts={accounts} currencies={me.currencies} baseCurrency={base} displayCurrency={display} onChanged={reload} onOpenReport={(id) => go({ kind: 'account', accountId: id })} onOpenSavings={() => go({ kind: 'savings' })} onOpenPortfolio={() => go({ kind: 'portfolio' })} />}
              <TabBar tab={tab} onTab={setTab} onAdd={onAdd} />
            </>
          )
          break
        default:
          body = null
      }

      return (
        <ThemeContext.Provider value={theme}>
          <PrivacyContext.Provider value={privacy}>
            {body}
            {overlay}
          </PrivacyContext.Provider>
        </ThemeContext.Provider>
      )
    }
  }
}
