import { useEffect, useState, type FormEvent } from 'react'
import { ApiError, createAccount, fetchPortfolio, type Account, type AccountKind, type Portfolio } from './api'
import { AccountFocus } from './AccountFocus'
import { holdings, totalBalance } from './accounts'
import { errorText, t } from './i18n'
import { Modal } from './Modal'
import { Money } from './MoneyText'
import { onKey } from './onKey'
import { parseAmount } from './money'
import { ChipPicker, CurrencyPicker } from './Pickers'

type Props = {
  accounts: Account[]
  currencies: string[]
  baseCurrency: string
  // Валюта, в которой показаны итоги на обороте карты.
  displayCurrency: string
  onChanged: () => void
  onOpenReport: (accountId: number) => void
  onOpenSavings: () => void
  onOpenPortfolio: () => void
}

const kindOptions = (['debit', 'credit', 'cash', 'wallet'] as const).map((k) => ({ value: k, label: t.accountKinds[k] }))

export function AccountsScreen({ accounts: every, currencies, baseCurrency, displayCurrency, onChanged, onOpenReport, onOpenSavings, onOpenPortfolio }: Props) {
  // Копилки живут на экране «Накопления», здесь только обычные счета и кошельки.
  const accounts = every.filter((a) => a.kind !== 'piggy')
  const [adding, setAdding] = useState(accounts.length === 0)
  const [focused, setFocused] = useState<number | null>(null)
  const focusedAccount = accounts.find((a) => a.id === focused)
  const total = totalBalance(every)
  const piggy = every.find((a) => a.kind === 'piggy' && !a.archived)
  const saved = piggy?.balances[0]?.balance_minor ?? 0
  const share = piggy?.target_minor ? Math.min(saved / piggy.target_minor, 1) : 0
  const RING = 2 * Math.PI * 14
  const [portfolio, setPortfolio] = useState<Portfolio | null>(null)

  // Плитка портфеля показывает стоимость и прибыль; без неё остаётся подпись и переход.
  useEffect(() => {
    let alive = true
    fetchPortfolio(displayCurrency)
      .then((p) => alive && setPortfolio(p))
      .catch(() => {})
    return () => {
      alive = false
    }
  }, [displayCurrency, every])
  const value = portfolio?.items.reduce((s, i) => s + i.value_minor, 0) ?? 0
  const cost = portfolio?.items.reduce((s, i) => s + i.cost_minor, 0) ?? 0
  const profitBp = cost > 0 ? Math.round(((value - cost) * 10000) / cost) : null

  return (
    <main className="screen">
      <header className="row">
        <h1>{t.accounts.title}</h1>
        <span className="row-tight">
          <button type="button" className="chip" onClick={() => setAdding(true)}>
            {t.accounts.add}
          </button>
        </span>
      </header>

      {accounts.length > 0 && (
        <div className="card ops-hero">
          <span className="label">{t.accounts.total}</span>
          {total === null ? <p className="hint tight">{t.accounts.noRates}</p> : <Money minor={total} currency={displayCurrency} />}
          <p className="hint tight">{t.accounts.totalNote}</p>
        </div>
      )}

      {accounts.length === 0 && <p className="hint">{t.accounts.empty}</p>}

      <div className="list account-strip">
        {accounts.map((a) => (
          <div
            key={a.id}
            className={`account account-${a.kind} account-tap`}
            role="button"
            tabIndex={0}
            onClick={() => setFocused(a.id)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault()
                setFocused(a.id)
              }
            }}
          >
            <div className="row">
              <span className="account-name">{a.name}</span>
              {a.is_default && <span className="badge">{t.accounts.defaultBadge}</span>}
            </div>
            {a.kind === 'wallet' && a.balances.length > 1 ? <WalletBody account={a} displayCurrency={displayCurrency} /> : <PlainBody account={a} />}
            <div className="row account-meta">
              <span>{t.accountKinds[a.kind]}</span>
              <span>{a.multi_currency ? t.accounts.many : a.currency}</span>
            </div>
          </div>
        ))}
      </div>

      <div className="bento bento-full">
        <div role="button" tabIndex={0} className="bt bt-piggy bt-tap" onClick={onOpenSavings} onKeyDown={onKey(onOpenSavings)}>
          <span className="bt-top">
            <span className="bt-cap">{t.accounts.savings}</span>
            {piggy !== undefined && piggy.target_minor !== null && (
              <svg width="36" height="36" viewBox="0 0 38 38" role="img" aria-label={`${Math.round(share * 100)}%`}>
                <circle cx="19" cy="19" r="14" fill="none" stroke="#fff" strokeWidth="6" />
                <circle cx="19" cy="19" r="14" fill="none" stroke="#8a6cff" strokeWidth="6" strokeLinecap="round" strokeDasharray={`${share * RING} ${RING}`} transform="rotate(-90 19 19)" />
              </svg>
            )}
          </span>
          {piggy === undefined ? (
            <b className="bt-empty">{t.fullTiles.noPiggy}</b>
          ) : (
            <>
              <span className="bt-sub">{piggy.name}</span>
              <Money minor={saved} currency={piggy.currency} />
            </>
          )}
        </div>
        <div role="button" tabIndex={0} className="bt bt-pay bt-tap" onClick={onOpenPortfolio} onKeyDown={onKey(onOpenPortfolio)}>
          <span className="bt-top">
            <span className="bt-cap">{t.accounts.portfolio}</span>
            {profitBp !== null && <span className="bt-badge">{`${profitBp >= 0 ? '+' : '−'}${(Math.abs(profitBp) / 100).toFixed(1).replace('.', ',')}%`}</span>}
          </span>
          {portfolio !== null && portfolio.items.length > 0 ? <Money minor={value} currency={portfolio.currency} /> : <b className="bt-empty">{t.fullTiles.noPortfolio}</b>}
        </div>
      </div>

      {focusedAccount !== undefined && (
        <AccountFocus
          account={focusedAccount}
          baseCurrency={displayCurrency}
          onReport={() => {
            setFocused(null)
            onOpenReport(focusedAccount.id)
          }}
          onClose={() => setFocused(null)}
        />
      )}

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

function PlainBody({ account }: { account: Account }) {
  return (
    <>
      <div className="account-balance">
        <Money minor={account.balances[0]?.balance_minor ?? 0} currency={account.currency} />
      </div>
      {account.balances.length > 1 && (
        <div className="account-others">
          {account.balances.slice(1).map((b) => (
            <Money key={b.currency} minor={b.balance_minor} currency={b.currency} />
          ))}
        </div>
      )}
    </>
  )
}

const WALLET_TOP = 3

// Кошелёк с несколькими монетами: общая стоимость в валюте показа и три крупнейших остатка, остальное числом.
function WalletBody({ account, displayCurrency }: { account: Account; displayCurrency: string }) {
  const h = holdings(account, WALLET_TOP)
  return (
    <>
      <div className="account-balance">
        {h.total === null ? <Money minor={account.balances[0]?.balance_minor ?? 0} currency={account.currency} /> : <Money minor={h.total} currency={displayCurrency} />}
      </div>
      <div className="account-others wallet-top">
        {h.top.map((b) => (
          <Money key={b.currency} minor={b.balance_minor} currency={b.currency} />
        ))}
        {h.more > 0 && (
          <span className="wallet-more">
            {t.accounts.more} {h.more}
          </span>
        )}
      </div>
    </>
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
  const [mode, setMode] = useState<'one' | 'many'>('one')
  const [opening, setOpening] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    const openingMinor = opening.trim() === '' ? 0 : parseAmount(opening, currency)
    if (openingMinor === null) return setError(errorText('invalid_amount'))

    setBusy(true)
    try {
      await createAccount(name, kind, currency, mode === 'many', openingMinor)
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
      <ChipPicker
        label={t.accounts.kind}
        value={kind}
        options={kindOptions}
        onChange={(k) => {
          setKind(k)
          // Кошелёк и биржа почти всегда держат несколько валют.
          setMode(k === 'wallet' ? 'many' : mode)
        }}
      />
      {kind === 'wallet' && <p className="hint tight">{t.accounts.walletHint}</p>}
      <ChipPicker
        label={t.accounts.currencyMode}
        value={mode}
        options={[
          { value: 'one' as const, label: t.accounts.one },
          { value: 'many' as const, label: t.accounts.many },
        ]}
        onChange={setMode}
      />
      {mode === 'many' && <p className="hint tight">{t.accounts.manyHint}</p>}
      <CurrencyPicker label={mode === 'many' ? t.accounts.mainCurrency : t.accounts.currency} value={currency} options={currencies} onChange={setCurrency} />
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
