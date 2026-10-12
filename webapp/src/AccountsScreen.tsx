import { useState, type FormEvent } from 'react'
import { ApiError, createAccount, type Account, type AccountKind } from './api'
import { errorText, t } from './i18n'
import { Modal } from './Modal'
import { Money } from './MoneyText'
import { parseAmount } from './money'
import { ChipPicker } from './Pickers'

type Props = {
  accounts: Account[]
  currencies: string[]
  baseCurrency: string
  onChanged: () => void
}

const kindOptions = (['debit', 'credit', 'cash'] as const).map((k) => ({ value: k, label: t.accountKinds[k] }))

export function AccountsScreen({ accounts, currencies, baseCurrency, onChanged }: Props) {
  const [adding, setAdding] = useState(accounts.length === 0)

  return (
    <main className="screen">
      <header className="row">
        <h1>{t.accounts.title}</h1>
        <button type="button" className="chip" onClick={() => setAdding(true)}>
          {t.accounts.add}
        </button>
      </header>

      {accounts.length === 0 && <p className="hint">{t.accounts.empty}</p>}

      <div className="list">
        {accounts.map((a) => (
          <article key={a.id} className={`account account-${a.kind}`}>
            <div className="row">
              <span className="account-name">{a.name}</span>
              {a.is_default && <span className="badge">{t.accounts.defaultBadge}</span>}
            </div>
            <div className="account-balance">
              <Money minor={a.balance_minor} currency={a.currency} />
            </div>
            <div className="row account-meta">
              <span>{t.accountKinds[a.kind]}</span>
              <span>{a.currency}</span>
            </div>
          </article>
        ))}
      </div>

      {adding && (
        <Modal title={t.accounts.add} onClose={() => setAdding(false)}>
          <AccountForm
            currencies={currencies}
            baseCurrency={baseCurrency}
            onDone={() => {
              setAdding(false)
              onChanged()
            }}
          />
        </Modal>
      )}
    </main>
  )
}

type FormProps = {
  currencies: string[]
  baseCurrency: string
  onDone: () => void
}

function AccountForm({ currencies, baseCurrency, onDone }: FormProps) {
  const [name, setName] = useState('')
  const [kind, setKind] = useState<AccountKind>('debit')
  const [currency, setCurrency] = useState(baseCurrency)
  const [opening, setOpening] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    const openingMinor = opening.trim() === '' ? 0 : parseAmount(opening)
    if (openingMinor === null) return setError(errorText('invalid_amount'))

    setBusy(true)
    try {
      await createAccount(name, kind, currency, openingMinor)
      onDone()
    } catch (err) {
      setError(errorText(err instanceof ApiError ? err.code : 'internal'))
      setBusy(false)
    }
  }

  return (
    <form className="form" onSubmit={submit}>
      <label className="field">
        <span className="label">{t.accounts.name}</span>
        <input value={name} maxLength={40} placeholder={t.accounts.namePlaceholder} autoFocus onChange={(e) => setName(e.target.value)} />
      </label>
      <ChipPicker label={t.accounts.kind} value={kind} options={kindOptions} onChange={setKind} />
      <ChipPicker label={t.accounts.currency} value={currency} options={currencies.map((c) => ({ value: c, label: c }))} onChange={setCurrency} />
      <label className="field">
        <span className="label">{t.accounts.opening}</span>
        <input inputMode="decimal" value={opening} placeholder="0,00" onChange={(e) => setOpening(e.target.value)} />
      </label>
      <p className="hint tight">{t.accounts.openingHint}</p>
      {error !== null && <p className="error">{error}</p>}
      <button type="submit" className="primary" disabled={busy || name.trim() === ''}>
        {t.save}
      </button>
    </form>
  )
}
