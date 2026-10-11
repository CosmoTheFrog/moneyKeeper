import { useState } from 'react'
import { ApiError, setBaseCurrency } from './api'
import { currencyName, errorText, t } from './i18n'

type Props = {
  currencies: string[]
  onDone: () => void
}

export function CurrencyScreen({ currencies, onDone }: Props) {
  const [picked, setPicked] = useState(currencies[0] ?? '')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit() {
    setBusy(true)
    setError(null)
    try {
      await setBaseCurrency(picked)
      onDone()
    } catch (e) {
      // 409: валюту уже выбрали, например в другом окне. Идём дальше.
      if (e instanceof ApiError && e.status === 409) {
        onDone()
        return
      }
      setError(errorText(e instanceof ApiError ? e.code : 'internal'))
      setBusy(false)
    }
  }

  return (
    <main className="screen">
      <div className="step">{t.currency.step}</div>
      <h1>{t.currency.title}</h1>
      <p className="hint">{t.currency.hint}</p>

      <div className="list" role="radiogroup" aria-label={t.currency.title}>
        {currencies.map((code) => (
          <button
            key={code}
            type="button"
            role="radio"
            aria-checked={code === picked}
            className={code === picked ? 'option option-picked' : 'option'}
            onClick={() => setPicked(code)}
          >
            <span className="option-sign">{code}</span>
            <span className="option-name">{currencyName(code)}</span>
          </button>
        ))}
      </div>

      {error !== null && <p className="error">{error}</p>}
      <button type="button" className="primary" disabled={busy || picked === ''} onClick={submit}>
        {t.currency.next}
      </button>
    </main>
  )
}
