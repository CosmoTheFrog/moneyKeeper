import { useEffect, useState } from 'react'
import { fetchMe } from './api'
import { t } from './i18n'

type State =
  | { kind: 'loading' }
  | { kind: 'ok'; name: string }
  | { kind: 'outside' }
  | { kind: 'error' }

export function App() {
  const [state, setState] = useState<State>({ kind: 'loading' })

  useEffect(() => {
    const webApp = window.Telegram?.WebApp
    if (webApp === undefined || webApp.initData === '') {
      setState({ kind: 'outside' })
      return
    }
    webApp.ready()
    fetchMe(webApp.initData)
      .then((me) => setState({ kind: 'ok', name: me.first_name }))
      .catch(() => setState({ kind: 'error' }))
  }, [])

  switch (state.kind) {
    case 'loading':
      return <p>{t.loading}</p>
    case 'ok':
      return <h1>{t.hello(state.name)}</h1>
    case 'outside':
      return <p>{t.openInTelegram}</p>
    case 'error':
      return <p>{t.authFailed}</p>
  }
}
