import { useContext, useState, type FormEvent } from 'react'
import { ApiError, createAccount, type Account, type AccountKind } from './api'
import { errorText, t } from './i18n'
import { Money, PrivacyContext } from './MoneyText'
import { parseAmount } from './money'

type Props = {
  accounts: Account[]
  currencies: string[]
  baseCurrency: string
  onChanged: () => void
}

const kinds: AccountKind[] = ['debit', 'credit', 'cash']

export function AccountsScreen({ accounts, currencies, baseCurrency, onChanged }: Props) {
  const { hidden, toggle } = useContext(PrivacyContext)
  const [adding, setAdding] = useState(accounts.length === 0)
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
      setName('')
      setOpening('')
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
        <span className="row-tight">
          <button type="button" className="chip" onClick={toggle}>
            {hidden ? t.privacy.show : t.privacy.hide}
          </button>
          {!adding && (
            <button type="button" className="chip" onClick={() => setAdding(true)}>
              {t.accounts.add}
            </button>
          )}
        </span>
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
          <label>
            {t.accounts.opening}
            <input inputMode="decimal" value={opening} placeholder="0,00" onChange={(e) => setOpening(e.target.value)} />
          </label>
          <p className="hint tight">{t.accounts.openingHint}</p>
          {error !== null && <p className="error">{error}</p>}
          <div className="row">
            <button type="submit" className="primary" disabled={busy || name.trim() === ''}>
              {t.save}
            </button>
            {accounts.length > 0 && (
              <button type="button" className="chip" onClick={() => setAdding(false)}>
                {t.cancel}
              </button>
            )}
          </div>
        </form>
      )}
    </main>
  )
}
