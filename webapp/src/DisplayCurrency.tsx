import { useEffect, useState } from 'react'
import { fetchRates, type RateTable } from './api'
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

type Loaded = { kind: 'loading' } | { kind: 'failed' } | { kind: 'ready'; table: RateTable }

function RatesModal({ display, currencies, onChange, onClose }: ModalProps) {
  const [loaded, setLoaded] = useState<Loaded>({ kind: 'loading' })

  useEffect(() => {
    let alive = true
    setLoaded({ kind: 'loading' })
    fetchRates(display)
      .then((table) => alive && setLoaded({ kind: 'ready', table }))
      .catch(() => alive && setLoaded({ kind: 'failed' }))
    return () => {
      alive = false
    }
  }, [display])

  return (
    <Modal title={t.rates.title} onClose={onClose}>
      <CurrencyPicker label={t.rates.countIn} value={display} options={currencies} onChange={onChange} />

      {loaded.kind === 'loading' && <p className="hint">{t.loading}</p>}
      {loaded.kind === 'failed' && <p className="error">{t.rates.failed}</p>}
      {loaded.kind === 'ready' && (
        <>
          <p className="hint tight">
            {t.rates.on} {loaded.table.day}
          </p>
          <div className="rates">
            {loaded.table.rates.map((r) => (
              <div key={r.currency} className="rate-row">
                <span className="rate-from">1 {currencySign(r.currency)}</span>
                <span className="rate-value">
                  {formatRate(r.rate)} {currencySign(loaded.table.in)}
                </span>
              </div>
            ))}
          </div>
          <p className="hint tight">{t.rates.note}</p>
        </>
      )}
    </Modal>
  )
}
