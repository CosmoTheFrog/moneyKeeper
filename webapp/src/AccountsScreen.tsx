import { useState, type FormEvent } from 'react'
import { ApiError, createAccount, type Account, type AccountKind } from './api'
import { errorText, t } from './i18n'

type Props = {
  accounts: Account[]
  currencies: string[]
  baseCurrency: string
  onChanged: () => void
}

const kinds: AccountKind[] = ['debit', 'credit', 'cash']

export function AccountsScreen({ accounts, currencies, baseCurrency, onChanged }: Props) {
  const [adding, setAdding] = useState(accounts.length === 0)
  const [name, setName] = useState('')
  const [kind, setKind] = useState<AccountKind>('debit')
  const [currency, setCurrency] = useState(baseCurrency)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await createAccount(name, kind, currency)
      setName('')
      setAdding(false)
      onChanged()
    } catch (err) {
      setError(errorText(err instanceof ApiError ? err.code : 'internal'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <main className="screen">
      <header className="row">
        <h1>{t.accounts.title}</h1>
        {!adding && (
          <button type="button" className="chip" onClick={() => setAdding(true)}>
            {t.accounts.add}
          </button>
        )}
      </header>

      {accounts.length === 0 && <p className="hint">{t.accounts.empty}</p>}

      <div className="list">
        {accounts.map((a) => (
          <article key={a.id} className={`account account-${a.kind}`}>
            <div className="row">
              <span className="account-name">{a.name}</span>
              {a.is_default && <span className="badge">{t.accounts.defaultBadge}</span>}
            </div>
            <div className="row account-meta">
              <span>{t.accountKinds[a.kind]}</span>
              <span>{a.currency}</span>
            </div>
          </article>
        ))}
      </div>

      {adding && (
        <form className="card" onSubmit={submit}>
          <label>
            {t.accounts.name}
            <input value={name} maxLength={40} placeholder={t.accounts.namePlaceholder} onChange={(e) => setName(e.target.value)} />
          </label>
          <label>
            {t.accounts.kind}
            <select value={kind} onChange={(e) => setKind(kinds.find((k) => k === e.target.value) ?? 'debit')}>
              {kinds.map((k) => (
                <option key={k} value={k}>
                  {t.accountKinds[k]}
                </option>
              ))}
            </select>
          </label>
          <label>
            {t.accounts.currency}
            <select value={currency} onChange={(e) => setCurrency(e.target.value)}>
              {currencies.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </select>
          </label>
          {error !== null && <p className="error">{error}</p>}
          <div className="row">
            <button type="submit" className="primary" disabled={busy || name.trim() === ''}>
              {t.accounts.save}
            </button>
            {accounts.length > 0 && (
              <button type="button" className="chip" onClick={() => setAdding(false)}>
                {t.accounts.cancel}
              </button>
            )}
          </div>
        </form>
      )}
    </main>
  )
}
