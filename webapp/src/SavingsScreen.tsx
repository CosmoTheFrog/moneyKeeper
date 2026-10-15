import { useState, type FormEvent } from 'react'
import { ApiError, createPiggy, createTransaction, type Account } from './api'
import { errorText, t } from './i18n'
import { Modal } from './Modal'
import { Money } from './MoneyText'
import { parseAmount } from './money'
import { CurrencyPicker } from './Pickers'
import { monthsUntil, perMonth, progressPercent } from './savings'

type Props = {
  accounts: Account[]
  allCurrencies: string[]
  today: string
  onBack: () => void
  onChanged: () => void
}

const RADIUS = 64
const CIRCLE = 2 * Math.PI * RADIUS

// Копилка: свинья с монетой. Монета падает в щель, когда копилку только что пополнили.
function Pig({ dropKey }: { dropKey: number }) {
  return (
    <svg className="pig" viewBox="0 0 120 100" role="img" aria-label={t.savings.title}>
      <ellipse cx="60" cy="58" rx="38" ry="27" fill="#f2a6b8" />
      <ellipse cx="97" cy="60" rx="11" ry="9" fill="#e98aa2" />
      <circle cx="94" cy="60" r="1.8" fill="#9c4a60" />
      <circle cx="100" cy="60" r="1.8" fill="#9c4a60" />
      <path d="M78 36 L88 22 L94 40 Z" fill="#e98aa2" />
      <circle cx="84" cy="50" r="2.6" fill="#4a2a35" />
      <rect x="34" y="78" width="10" height="14" rx="4" fill="#e98aa2" />
      <rect x="72" y="78" width="10" height="14" rx="4" fill="#e98aa2" />
      <path d="M22 54 q-12 -4 -8 -14" stroke="#e98aa2" strokeWidth="4" fill="none" strokeLinecap="round" />
      <rect x="48" y="33" width="22" height="4" rx="2" fill="#9c4a60" />
      <g key={dropKey} className={dropKey > 0 ? 'coin coin-drop' : 'coin'}>
        <circle cx="59" cy="12" r="9" fill="#f4c542" stroke="#c99a12" strokeWidth="2" />
        <text x="59" y="16" textAnchor="middle" fontSize="11" fontWeight="800" fill="#8a6508">
          ₾
        </text>
      </g>
    </svg>
  )
}

// Накопления: копилки с целью, прогрессом и пополнением. Пополнение каждый раз спрашивает, с какого счёта списать.
export function SavingsScreen({ accounts, allCurrencies, today, onBack, onChanged }: Props) {
  const piggies = accounts.filter((a) => a.kind === 'piggy')
  const [selectedId, setSelectedId] = useState(0)
  const [adding, setAdding] = useState(false)
  const [topping, setTopping] = useState(false)
  const [dropKey, setDropKey] = useState(0)

  const selected = piggies.find((p) => p.id === selectedId) ?? piggies[0]
  const saved = selected?.balances[0]?.balance_minor ?? 0
  const target = selected?.target_minor ?? null
  const percent = progressPercent(saved, target)
  const remaining = target === null ? 0 : Math.max(target - saved, 0)
  const months = selected?.target_date != null ? monthsUntil(today, selected.target_date) : 0

  return (
    <main className="screen">
      <header className="row">
        <button type="button" className="chip" onClick={onBack}>
          {t.back}
        </button>
        <button type="button" className="chip" onClick={() => setAdding(true)}>
          {t.savings.add}
        </button>
      </header>
      <h1>{t.savings.title}</h1>

      {selected === undefined && <p className="hint">{t.savings.empty}</p>}

      {selected !== undefined && (
        <div className="card piggy-hero">
          <div className="piggy-ring">
            <svg viewBox="0 0 160 160" aria-hidden="true">
              <g transform="rotate(-90 80 80)">
                <circle cx="80" cy="80" r={RADIUS} fill="none" style={{ stroke: 'var(--line)' }} strokeWidth="12" />
                <circle cx="80" cy="80" r={RADIUS} fill="none" stroke="#e98aa2" strokeWidth="12" strokeLinecap="round" strokeDasharray={`${(CIRCLE * percent) / 100} ${CIRCLE}`} />
              </g>
            </svg>
            <Pig dropKey={dropKey} />
          </div>
          <h2 className="piggy-name">{selected.name}</h2>
          <p className="piggy-saved">
            <Money minor={saved} currency={selected.currency} />
            {target !== null && (
              <>
                {' '}
                {t.savings.of} <Money minor={target} currency={selected.currency} />
              </>
            )}
          </p>
          <p className="hint tight">
            {target === null && t.savings.noGoal}
            {target !== null && remaining === 0 && t.savings.reached}
            {target !== null && remaining > 0 && (
              <>
                {t.savings.left} <Money minor={remaining} currency={selected.currency} />
                {months > 0 && (
                  <>
                    {' · '}
                    {t.savings.perMonth} <Money minor={perMonth(remaining, months)} currency={selected.currency} />
                  </>
                )}
              </>
            )}
          </p>
          <button type="button" className="primary" onClick={() => setTopping(true)}>
            {t.savings.topUp}
          </button>
        </div>
      )}

      {piggies.length > 1 && (
        <div className="list">
          {piggies.map((p) => (
            <button key={p.id} type="button" className={p.id === selected?.id ? 'choice choice-on' : 'choice'} onClick={() => setSelectedId(p.id)}>
              <span>{p.name}</span>
              <Money minor={p.balances[0]?.balance_minor ?? 0} currency={p.currency} />
            </button>
          ))}
        </div>
      )}

      {topping && selected !== undefined && (
        <Modal title={`${t.savings.topUpTitle} «${selected.name}»`} onClose={() => setTopping(false)}>
          <TopUpForm
            piggy={selected}
            sources={accounts.filter((a) => !a.off_budget && (a.multi_currency || a.currency === selected.currency))}
            onDone={() => {
              setTopping(false)
              setDropKey((k) => k + 1)
              onChanged()
            }}
          />
        </Modal>
      )}

      {adding && (
        <Modal title={t.savings.add} onClose={() => setAdding(false)}>
          <PiggyForm
            currencies={allCurrencies}
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

type TopUpProps = { piggy: Account; sources: Account[]; onDone: () => void }

function TopUpForm({ piggy, sources, onDone }: TopUpProps) {
  const [amount, setAmount] = useState('')
  const [sourceId, setSourceId] = useState(0)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    const minor = parseAmount(amount, piggy.currency)
    if (minor === null) return setError(errorText('invalid_amount'))
    if (sourceId === 0) return setError(t.savings.noSource)
    if (piggy.category_id === null) return setError(errorText('internal'))
    setBusy(true)
    try {
      // Пополнение это перевод в конверт копилки: деньги уходят из бюджета и сразу видны в копилке.
      await createTransaction({
        kind: 'transfer',
        account_id: sourceId,
        to_account_id: piggy.id,
        category_id: piggy.category_id,
        amount_minor: minor,
        currency: piggy.currency,
        note: '',
      })
      onDone()
    } catch (err) {
      setError(errorText(err instanceof ApiError ? err.code : 'internal'))
      setBusy(false)
    }
  }

  return (
    <form className="form" onSubmit={submit}>
      <label className="field">
        <span className="label">{t.savings.amount}</span>
        <input inputMode="decimal" value={amount} placeholder="0,00" autoFocus onChange={(e) => setAmount(e.target.value)} />
      </label>
      <span className="label">{t.savings.from}</span>
      <div className="choices">
        {sources.map((a) => (
          <button key={a.id} type="button" className={a.id === sourceId ? 'choice choice-on' : 'choice'} onClick={() => setSourceId(a.id)}>
            <span>{a.name}</span>
            <Money minor={a.balances.find((b) => b.currency === piggy.currency)?.balance_minor ?? 0} currency={piggy.currency} />
          </button>
        ))}
        {sources.length === 0 && <p className="hint tight">{t.savings.noSource}</p>}
      </div>
      <p className="hint tight">{t.savings.fromHint}</p>
      <p className="hint tight">{t.savings.moves}</p>
      {error !== null && <p className="error">{error}</p>}
      <button type="submit" className="primary" disabled={busy}>
        {t.savings.topUp}
      </button>
    </form>
  )
}

function PiggyForm({ currencies, onDone }: { currencies: string[]; onDone: () => void }) {
  const [name, setName] = useState('')
  const [currency, setCurrency] = useState(currencies[0] ?? '')
  const [goal, setGoal] = useState('')
  const [until, setUntil] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    const target = goal.trim() === '' ? null : parseAmount(goal, currency)
    if (goal.trim() !== '' && target === null) return setError(errorText('invalid_amount'))
    setBusy(true)
    try {
      await createPiggy(name, currency, target, until === '' ? null : until)
      onDone()
    } catch (err) {
      setError(errorText(err instanceof ApiError ? err.code : 'internal'))
      setBusy(false)
    }
  }

  return (
    <form className="form" onSubmit={submit}>
      <label className="field">
        <span className="label">{t.savings.name}</span>
        <input value={name} maxLength={40} placeholder={t.savings.namePlaceholder} autoFocus onChange={(e) => setName(e.target.value)} />
      </label>
      <CurrencyPicker label={t.accounts.currency} value={currency} options={currencies} onChange={setCurrency} />
      <label className="field">
        <span className="label">{t.savings.goal}</span>
        <input inputMode="decimal" value={goal} placeholder="0,00" onChange={(e) => setGoal(e.target.value)} />
      </label>
      <label className="field">
        <span className="label">{t.savings.goalDate}</span>
        <input type="date" value={until} onChange={(e) => setUntil(e.target.value)} />
      </label>
      {error !== null && <p className="error">{error}</p>}
      <button type="submit" className="primary" disabled={busy || name.trim() === ''}>
        {t.save}
      </button>
    </form>
  )
}
