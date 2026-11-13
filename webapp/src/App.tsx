import { lazy, Suspense, useCallback, useEffect, useMemo, useState } from 'react'
import { AccountsScreen } from './AccountsScreen'
import {
  deleteTransaction,
  fetchAccounts,
  fetchCategories,
  fetchMe,
  fetchTransactions,
  type Account,
  type Category,
  type Me,
  type Transaction,
} from './api'
import { BudgetScreen } from './BudgetScreen'
import { CategoryScreen } from './CategoryScreen'
import { DayScreen } from './DayScreen'
import { ModeScreen } from './ModeScreen'
import { CategoryColorContext, categoryColors } from './categoryColor'
import { CurrencyScreen } from './CurrencyScreen'
import { TourScreen } from './TourScreen'
import { EntryForm } from './EntryForm'
import { t } from './i18n'
import { dayIn, toEntries, type Entry } from './ledger'
import { NoPrivacy, PrivacyContext } from './MoneyText'
import { isCoin, isStock, registerCurrencies } from './money'
import { MoreScreen } from './MoreScreen'
import { OperationsScreen } from './OperationsScreen'
import { SwipeRow } from './SwipeRow'
import { haptic } from './haptic'
import { ErrorState, ScreenSkeleton } from './States'
import { ReportsScreen } from './ReportsScreen'
import { TabBar, type Tab } from './TabBar'
import { readEnabled, saveEnabled, visibleCurrencies } from './currencyChoice'
import { applyTheme, readTheme, saveTheme, type Theme } from './theme'
import { ThemeContext } from './ThemeSettings'

// Экраны, которые открывают редко, грузятся отдельными кусками при первом открытии.
const AccountReport = lazy(() => import('./AccountReport').then((m) => ({ default: m.AccountReport })))
const PortfolioScreen = lazy(() => import('./PortfolioScreen').then((m) => ({ default: m.PortfolioScreen })))
const SchedulesScreen = lazy(() => import('./SchedulesScreen').then((m) => ({ default: m.SchedulesScreen })))
const SavingsScreen = lazy(() => import('./SavingsScreen').then((m) => ({ default: m.SavingsScreen })))

type Data = {
  me: Me
  accounts: Account[]
  categories: Category[]
  transactions: Transaction[]
}

type State = { kind: 'loading' } | { kind: 'outside' } | { kind: 'error' } | { kind: 'ready'; data: Data }

type Screen =
  | { kind: 'tabs' }
  | { kind: 'add'; categoryId?: number; presetDate?: string; returnTo: Screen }
  | { kind: 'edit'; entry: Entry; returnTo: Screen }
  | { kind: 'category'; categoryId: number }
  | { kind: 'day'; date: string }
  | { kind: 'more' }
  | { kind: 'account'; accountId: number }
  | { kind: 'savings'; back?: Tab }
  | { kind: 'schedules'; back?: Tab }
  | { kind: 'portfolio'; back?: Tab }

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
  const [themeName, setThemeName] = useState<Theme>(readTheme)
  const [displayChoice, setDisplayChoice] = useState(readDisplay)
  const [enabledCurrencies, setEnabledCurrencies] = useState(readEnabled)

  const theme = useMemo(
    () => ({
      theme: themeName,
      setTheme: (next: Theme) => {
        saveTheme(next)
        setThemeName(next)
      },
    }),
    [themeName],
  )

  // Оформление применяется к странице целиком.
  useEffect(() => applyTheme(themeName), [themeName])

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
      // Категории от профиля не зависят, их запрос идёт параллельно с ним.
      const structure = fetchCategories()
      structure.catch(() => {}) // ошибку разберёт await ниже; без этого при раннем выходе она осталась бы необработанной
      const me = await fetchMe()
      registerCurrencies(me.currency_info)
      if (me.budget.base_currency === null) {
        setState({ kind: 'ready', data: { me, accounts: [], categories: [], transactions: [] } })
        return
      }
      // Сохранённая валюта могла пропасть из списка: тогда считаем в базовой.
      const target = me.currencies.includes(displayChoice) ? displayChoice : ''
      const [categories, accounts, transactions] = await Promise.all([structure, fetchAccounts(target), fetchTransactions(target)])
      setState({ kind: 'ready', data: { me, accounts, categories, transactions } })
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
      return <ScreenSkeleton />
    case 'outside':
      return <p className="screen hint">{t.openInTelegram}</p>
    case 'error':
      return <ErrorState onRetry={() => void load()} />
    case 'ready': {
      const { me, accounts: everyAccount, categories } = state.data
      // Закрытые счета не показываются в списках и выборе, но остаются в названиях старых записей.
      const openAccounts = everyAccount.filter((a) => !a.archived)
      // Простой режим: один скрытый основной счёт, выбирать счёт и переводить деньги не нужно.
      const simple = me.budget.interface_mode === 'simple'
      const accounts = simple ? openAccounts.filter((a) => a.is_default) : openAccounts
      const base = me.budget.base_currency
      const display = me.currencies.includes(displayChoice) ? displayChoice : (base ?? '')
      // Валюты для выбора: включённые в «Ещё → Мои валюты» плюс те, что уже нужны (база, показ, счета).
      const inUse = [base ?? '', display, ...everyAccount.flatMap((a) => [a.currency, ...a.balances.map((b) => b.currency)])]
      const shown = visibleCurrencies(me.currencies, enabledCurrencies, inUse, (c) => !isCoin(c) && !isStock(c))
      const changeEnabled = (list: string[]) => {
        saveEnabled(list)
        setEnabledCurrencies(list)
      }
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
      if (me.budget.interface_mode === null) {
        return <ModeScreen onDone={() => void load()} />
      }
      if (!me.tour_seen) {
        return <TourScreen simple={me.budget.interface_mode === 'simple'} onDone={() => void load()} />
      }

      const today = dayIn(me.timezone)
      const colors = categoryColors(categories)
      const reload = () => void load()
      const go = (to: Screen) => setScreen(to)
      const home: Screen = { kind: 'tabs' }
      // Вкладки «Счета» в простом режиме нет, «Платежи» в полном: если пользователь был на ней, показываем операции.
      const shownTab: Tab = (simple && tab === 'accounts') || (!simple && tab === 'schedules') ? 'operations' : tab

      const openEntry = (entry: Entry, returnTo: Screen) => go({ kind: 'edit', entry, returnTo })
      // Запись меняет только счета (остатки) и список записей; профиль и категории при этом не трогаются.
      const refreshLedger = async () => {
        try {
          const target = me.currencies.includes(displayChoice) ? displayChoice : ''
          const [nextAccounts, transactions] = await Promise.all([fetchAccounts(target), fetchTransactions(target)])
          setState((prev) => (prev.kind === 'ready' ? { kind: 'ready', data: { ...prev.data, accounts: nextAccounts, transactions } } : prev))
        } catch {
          reload()
        }
      }
      const finish = (returnTo: Screen) => {
        haptic('success')
        go(returnTo)
        void refreshLedger()
      }
      // Удаление смахиванием: запись и её вторая нога удаляются на сервере, список перечитывается.
      const removeEntry = async (entry: Entry) => {
        try {
          await deleteTransaction(entry.id)
        } catch {
          // не удалилось: список ниже перечитается и покажет, что запись на месте
        }
        await refreshLedger()
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
            allAccounts={everyAccount}
            categories={categories}
            allCurrencies={me.currencies}
            simple={simple}
            recentEntries={entries}
            onSaved={() => void refreshLedger()}
            presetCategoryId={screen.categoryId}
            presetDate={screen.presetDate}
            onDone={() => finish(screen.returnTo)}
            onCancel={() => go(screen.returnTo)}
          />
        )
      } else if (screen.kind === 'edit') {
        overlay = (
          <EntryForm
            accounts={accounts}
            allAccounts={everyAccount}
            categories={categories}
            allCurrencies={me.currencies}
            simple={simple}
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
              entries={entries.filter((e) => e.categoryId === category.id)}
              accounts={everyAccount}
              categories={categories}
              today={today}
              currency={display}
              onBack={() => go(home)}
              onAdd={() => go({ kind: 'add', categoryId: category.id, returnTo: here })}
              onOpen={(entry) => openEntry(entry, here)}
            />
          )
          break
        }
        case 'more':
          body = (
            <MoreScreen
              name={me.first_name}
              baseCurrency={base}
              accounts={accounts.filter((a) => a.kind !== 'piggy')}
              interfaceMode={me.budget.interface_mode}
              currencies={shown}
              registry={me.currencies}
              enabled={enabledCurrencies}
              pinned={inUse}
              onEnabled={changeEnabled}
              display={display}
              reminderHour={me.reminder_hour}
              onDisplay={changeDisplay}
              onChanged={reload}
              onBack={() => go(home)}
            />
          )
          break
        case 'day': {
          const here: Screen = { kind: 'day', date: layer.date }
          body = (
            <DayScreen
              date={layer.date}
              entries={entries}
              accounts={everyAccount}
              categories={categories}
              today={today}
              display={display}
              onBack={() => go(home)}
              onShift={(date) => go({ kind: 'day', date })}
              onOpen={(entry) => openEntry(entry, here)}
              onAdd={(date) => go({ kind: 'add', presetDate: date, returnTo: here })}
            />
          )
          break
        }
        case 'account': {
          const account = everyAccount.find((a) => a.id === layer.accountId)
          if (account === undefined) {
            body = null
            break
          }
          const here: Screen = { kind: 'account', accountId: account.id }
          body = (
            <AccountReport
              account={account}
              entries={entries}
              accounts={everyAccount}
                categories={categories}
              baseCurrency={display}
              currencies={shown}
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
        case 'schedules':
          body = (
            <SchedulesScreen
                categories={categories}
              accounts={accounts}
              baseCurrency={base}
              currencies={shown}
              today={today}
              ledger={state.data.transactions}
              onBack={() => {
                setTab(layer.back ?? 'budget')
                go(home)
              }}
              onAddExpense={(categoryId) => go({ kind: 'add', categoryId, returnTo: layer })}
            />
          )
          break
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
              currencies={shown}
              wallets={accounts.filter((a) => a.kind === 'wallet')}
              today={today}
              onChanged={reload}
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
              {shownTab !== 'operations' && (
                <button type="button" className="chip back-float" onClick={() => setTab('operations')}>
                  {t.back}
                </button>
              )}
              {shownTab === 'operations' && <OperationsScreen entries={entries} accounts={everyAccount} categories={categories} today={today} display={display} base={base} onOpen={(entry) => openEntry(entry, home)} onAdd={onAdd} onOpenDay={(date) => go({ kind: 'day', date })} onOpenMore={() => go({ kind: 'more' })} onOpenTab={setTab} onOpenCategory={(id) => go({ kind: 'category', categoryId: id })} onOpenSchedules={() => go({ kind: 'schedules', back: 'operations' })} onOpenSavings={() => go({ kind: 'savings', back: 'operations' })} name={me.first_name} simple={simple} wrapRow={(entry, row) => <SwipeRow onEdit={() => openEntry(entry, home)} onDelete={() => void removeEntry(entry)}>{row}</SwipeRow>} />}
              {shownTab === 'budget' && (
                <BudgetScreen
                        categories={categories}
                  accounts={accounts}
                  baseCurrency={base}
                  today={today}
                  envelopesEnabled={me.budget.envelopes_enabled}
                  ledger={state.data.transactions}
                  onOpenSchedules={() => go({ kind: 'schedules' })}
                  onChanged={reload}
                  onOpenCategory={(id) => go({ kind: 'category', categoryId: id })}
                />
              )}
              {shownTab === 'reports' && (
                <ReportsScreen
                  entries={entries}
                  accounts={everyAccount}
                        categories={categories}
                  baseCurrency={display}
                  currencies={shown}
                  onDisplay={changeDisplay}
                  today={today}
                  onOpen={(entry) => openEntry(entry, home)}
                />
              )}
              {shownTab === 'schedules' && (
                <SchedulesScreen
                  categories={categories}
                  accounts={accounts}
                  baseCurrency={base}
                  currencies={shown}
                  today={today}
                  ledger={state.data.transactions}
                  onAddExpense={(categoryId) => go({ kind: 'add', categoryId, returnTo: home })}
                />
              )}
              {shownTab === 'accounts' && !simple && <AccountsScreen accounts={accounts} currencies={shown} baseCurrency={base} displayCurrency={display} onDisplay={changeDisplay} onChanged={reload} onOpenReport={(id) => go({ kind: 'account', accountId: id })} onOpenSavings={() => go({ kind: 'savings' })} onOpenPortfolio={() => go({ kind: 'portfolio' })} />}
              <TabBar tab={shownTab} onTab={setTab} onAdd={onAdd} simple={simple} />
            </>
          )
          break
        default:
          body = null
      }

      // Режим скрытия сумм работает только на вкладках «Операции» и «Счета».
      const hidesSums = layer.kind === 'tabs' && (shownTab === 'operations' || shownTab === 'accounts')
      return (
        <ThemeContext.Provider value={theme}>
          <PrivacyContext.Provider value={privacy}>
            <CategoryColorContext.Provider value={colors}>
              <Suspense fallback={<ScreenSkeleton />}>
                {hidesSums || layer.kind === 'category' || layer.kind === 'day' || layer.kind === 'more' || layer.kind === 'add' || layer.kind === 'edit' ? body : <NoPrivacy>{body}</NoPrivacy>}
              </Suspense>
              {overlay}
            </CategoryColorContext.Provider>
          </PrivacyContext.Provider>
        </ThemeContext.Provider>
      )
    }
  }
}
