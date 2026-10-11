import { useCallback, useEffect, useState } from 'react'
import { AccountsScreen } from './AccountsScreen'
import { fetchAccounts, fetchMe, type Account, type Me } from './api'
import { CurrencyScreen } from './CurrencyScreen'
import { t } from './i18n'

type State =
  | { kind: 'loading' }
  | { kind: 'outside' }
  | { kind: 'error' }
  | { kind: 'ready'; me: Me; accounts: Account[] }

export function App() {
  const [state, setState] = useState<State>({ kind: 'loading' })

  const load = useCallback(async () => {
    try {
      const me = await fetchMe()
      const accounts = me.budget.base_currency === null ? [] : await fetchAccounts()
      setState({ kind: 'ready', me, accounts })
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
      const base = state.me.budget.base_currency
      if (base === null) {
        return <CurrencyScreen currencies={state.me.base_currencies} onDone={() => void load()} />
      }
      return <AccountsScreen accounts={state.accounts} currencies={state.me.currencies} baseCurrency={base} onChanged={() => void load()} />
    }
  }
}
