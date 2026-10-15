import { useState } from 'react'
import { fetchRates } from './api'
import { t } from './i18n'
import { Modal } from './Modal'
import { currencySign, formatRate } from './money'
import { CurrencyPicker } from './Pickers'

type Props = {
  display: string
  currencies: string[]
  onChange: (code: string) => void
}

// Плашка «В ₾»: показывает, в какой валюте считаются суммы, и по нажатию открывает выбор валюты с курсами.
export function DisplayCurrency({ display, currencies, onChange }: Props) {
  const [open, setOpen] = useState(false)

  return (
    <>
      <button type="button" className="chip chip-display" onClick={() => setOpen(true)}>
        {t.rates.chip} {currencySign(display)}
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          <path d="M6 9l6 6 6-6" />
        </svg>
      </button>
      {open && <RatesModal display={display} currencies={currencies} onChange={onChange} onClose={() => setOpen(false)} />}
    </>
  )
}

type ModalProps = Props & { onClose: () => void }

type Result = { kind: 'idle' } | { kind: 'loading' } | { kind: 'failed' } | { kind: 'ready'; day: string; rate: string | null }

function RatesModal({ display, currencies, onChange, onClose }: ModalProps) {
  const [from, setFrom] = useState(currencies.find((c) => c !== display) ?? display)
  const [to, setTo] = useState(display)
  const [result, setResult] = useState<Result>({ kind: 'idle' })

  function pick(set: (code: string) => void) {
    return (code: string) => {
      set(code)
      setResult({ kind: 'idle' })
    }
  }

  async function show() {
    setResult({ kind: 'loading' })
    try {
      const table = await fetchRates(to)
      setResult({ kind: 'ready', day: table.day, rate: from === to ? '1' : (table.rates.find((r) => r.currency === from)?.rate ?? null) })
    } catch {
      setResult({ kind: 'failed' })
    }
  }

  return (
    <Modal title={t.rates.title} onClose={onClose}>
      <CurrencyPicker label={t.rates.countIn} value={display} options={currencies} onChange={onChange} />
      <p className="hint tight">{t.rates.countInHint}</p>

      <h3 className="rates-head">{t.rates.rate}</h3>
      <CurrencyPicker label={t.rates.from} value={from} options={currencies} onChange={pick(setFrom)} />
      <CurrencyPicker label={t.rates.to} value={to} options={currencies} onChange={pick(setTo)} />
      <button type="button" className="primary" disabled={result.kind === 'loading'} onClick={() => void show()}>
        {t.rates.show}
      </button>

      {result.kind === 'loading' && <p className="hint">{t.loading}</p>}
      {result.kind === 'failed' && <p className="error">{t.rates.failed}</p>}
      {result.kind === 'ready' && (
        <div className="rate-result">
          {result.rate === null ? (
            <p className="error">{t.rates.failed}</p>
          ) : (
            <>
              <span className="rate-from">1 {currencySign(from)} =</span>
              <span className="rate-value">
                {formatRate(result.rate)} {currencySign(to)}
              </span>
              <p className="hint tight">
                {t.rates.on} {result.day}
              </p>
            </>
          )}
        </div>
      )}
      <p className="hint tight">{t.rates.note}</p>
    </Modal>
  )
}