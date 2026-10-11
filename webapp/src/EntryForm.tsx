import { useState, type FormEvent } from 'react'
import {
  ApiError,
  createTransaction,
  deleteTransaction,
  fetchEvents,
  parseQuick,
  updateTransaction,
  type Account,
  type Category,
  type CategoryGroup,
  type TransactionPatch,
  type TxEvent,
} from './api'
import { errorText, t } from './i18n'
import type { Entry } from './ledger'
import { Money } from './MoneyText'
import { formatInput, formatMinor, parseAmount } from './money'

type FormKind = 'expense' | 'income' | 'transfer' | 'opening'

type Props = {
  accounts: Account[]
  categories: Category[]
  // Без entry форма создаёт запись, с entry правит её.
  entry?: Entry
  onDone: () => void
  onCancel: () => void
}

const groupOrder: CategoryGroup[] = ['critical', 'debt', 'daily', 'rainy', 'goal']
const newKinds: FormKind[] = ['expense', 'income', 'transfer']

function kindLabel(kind: FormKind): string {
  return kind === 'opening' ? t.ledger.opening : t.kinds[kind]
}

export function EntryForm({ accounts, categories, entry, onDone, onCancel }: Props) {
  const editing = entry !== undefined
  const defaultAccount = accounts.find((a) => a.is_default) ?? accounts[0]

  const [kind, setKind] = useState<FormKind>(entry?.kind ?? 'expense')
  const [quick, setQuick] = useState('')
  const [understood, setUnderstood] = useState<string | null>(null)
  const [amount, setAmount] = useState(entry === undefined ? '' : formatInput(entry.amountMinor))
  const [accountId, setAccountId] = useState(entry?.accountId ?? defaultAccount?.id ?? 0)
  const [toAccountId, setToAccountId] = useState(entry?.toAccountId ?? 0)
  const [categoryId, setCategoryId] = useState(entry?.categoryId ?? 0)
  const [date, setDate] = useState(entry?.date ?? '')
  const [note, setNote] = useState(entry?.note ?? '')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [events, setEvents] = useState<TxEvent[] | null>(null)

  const account = accounts.find((a) => a.id === accountId)
  const currency = account?.currency ?? entry?.currency ?? ''
  const accountName = new Map(accounts.map((a) => [a.id, a.name]))
  const categoryName = new Map(categories.map((c) => [c.id, c.name]))

  // В правке и при переводе счета выбираются только среди счетов той же валюты.
  const sameCurrency = accounts.filter((a) => a.currency === currency)
  const canPickAccount = editing ? kind === 'expense' || kind === 'income' : true
  const canPickTarget = !editing && kind === 'transfer'
  const targets = sameCurrency.filter((a) => a.id !== accountId)

  function pickAccount(id: number) {
    setAccountId(id)
    if (id === toAccountId) setToAccountId(0)
  }

  async function understand() {
    const text = quick.trim()
    if (text === '') return
    setError(null)
    try {
      const p = await parseQuick(text)
      setAmount(formatInput(p.amount_minor))
      if (p.note !== '') setNote(p.note)
      let cur = currency
      if (p.currency !== '' && p.currency !== currency) {
        const match = accounts.find((a) => a.currency === p.currency)
        if (match === undefined) {
          setError(t.add.noAccountInCurrency)
        } else {
          pickAccount(match.id)
        }
        cur = p.currency
      }
      setUnderstood(`${formatMinor(p.amount_minor, cur)}${p.note !== '' ? ` · ${p.note}` : ''}`)
    } catch (e) {
      setUnderstood(null)
      setError(errorText(e instanceof ApiError ? e.code : 'internal'))
    }
  }

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    const minor = parseAmount(amount)
    if (minor === null) return setError(errorText('invalid_amount'))
    if (!editing && kind === 'expense' && categoryId === 0) return setError(t.add.pickCategory)
    if (canPickTarget && toAccountId === 0) return setError(t.add.noTarget)

    setBusy(true)
    try {
      if (entry === undefined) {
        if (kind === 'opening') throw new ApiError('invalid_transaction', 400)
        await createTransaction({
          kind,
          account_id: accountId,
          to_account_id: kind === 'transfer' ? toAccountId : undefined,
          category_id: kind === 'expense' ? categoryId : undefined,
          amount_minor: minor,
          date: date === '' ? undefined : date,
          note,
        })
      } else {
        const patch: TransactionPatch = {}
        if (minor !== Math.abs(entry.amountMinor)) patch.amount_minor = minor
        if (date !== entry.date) patch.date = date
        if (note !== entry.note) patch.note = note
        if (canPickAccount && accountId !== entry.accountId) patch.account_id = accountId
        if (kind === 'expense' && categoryId !== 0 && categoryId !== entry.categoryId) patch.category_id = categoryId
        if (Object.keys(patch).length > 0) await updateTransaction(entry.id, patch)
      }
      onDone()
    } catch (err) {
      setError(errorText(err instanceof ApiError ? err.code : 'internal'))
      setBusy(false)
    }
  }

  async function remove() {
    if (entry === undefined) return
    if (!confirmDelete) return setConfirmDelete(true)
    setBusy(true)
    try {
      await deleteTransaction(entry.id)
      onDone()
    } catch (err) {
      setError(errorText(err instanceof ApiError ? err.code : 'internal'))
      setBusy(false)
    }
  }

  async function toggleHistory() {
    if (entry === undefined) return
    if (events !== null) return setEvents(null)
    try {
      setEvents(await fetchEvents(entry.id))
    } catch (err) {
      setError(errorText(err instanceof ApiError ? err.code : 'internal'))
    }
  }

  return (
    <main className="screen">
      <header className="row">
        <button type="button" className="chip" onClick={onCancel}>
          {t.cancel}
        </button>
        <h1 className="h1-small">{editing ? t.edit.title : t.add.title}</h1>
        <span className="spacer" />
      </header>

      {!editing && (
        <div className="segmented" role="tablist">
          {newKinds.map((k) => (
            <button key={k} type="button" role="tab" aria-selected={kind === k} className={kind === k ? 'seg seg-on' : 'seg'} onClick={() => setKind(k)}>
              {kindLabel(k)}
            </button>
          ))}
        </div>
      )}
      {editing && <p className="hint">{kindLabel(kind)}</p>}

      <form className="stack" onSubmit={submit}>
        {!editing && (
          <div className="card">
            <label>
              {t.add.quick}
              <input
                value={quick}
                placeholder={t.add.quickPlaceholder}
                onChange={(e) => setQuick(e.target.value)}
                onBlur={() => void understand()}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault()
                    void understand()
                  }
                }}
              />
            </label>
            {understood !== null && (
              <p className="hint tight">
                {t.add.understood}: {understood}
              </p>
            )}
          </div>
        )}

        <div className="card">
          <label>
            {t.add.amount}
            <span className="amount-row">
              <input inputMode="decimal" value={amount} placeholder="0,00" onChange={(e) => setAmount(e.target.value)} />
              <span className="amount-cur">{currency}</span>
            </span>
          </label>
        </div>

        {canPickAccount && (
          <div className="card">
            <span className="label">{canPickTarget ? t.add.fromAccount : t.add.account}</span>
            <div className="choices">
              {(editing ? sameCurrency : accounts).map((a) => (
                <button key={a.id} type="button" className={a.id === accountId ? 'choice choice-on' : 'choice'} onClick={() => pickAccount(a.id)}>
                  <span>{a.name}</span>
                  <span className="choice-meta">{a.currency}</span>
                </button>
              ))}
            </div>
          </div>
        )}

        {canPickTarget && (
          <div className="card">
            <span className="label">{t.add.toAccount}</span>
            <div className="choices">
              {targets.map((a) => (
                <button key={a.id} type="button" className={a.id === toAccountId ? 'choice choice-on' : 'choice'} onClick={() => setToAccountId(a.id)}>
                  <span>{a.name}</span>
                  <span className="choice-meta">{a.currency}</span>
                </button>
              ))}
              {targets.length === 0 && <p className="hint tight">{t.add.noTarget}</p>}
            </div>
            <p className="hint tight">{t.add.transferHint}</p>
          </div>
        )}

        {kind === 'expense' && (
          <div className="card">
            <label>
              {t.add.category}
              <select value={categoryId} onChange={(e) => setCategoryId(Number(e.target.value))}>
                <option value={0} disabled={editing}>
                  {t.add.pickCategory}
                </option>
                {groupOrder.map((g) => {
                  const inGroup = categories.filter((c) => c.group === g && (!c.hidden || c.id === entry?.categoryId))
                  return inGroup.length === 0 ? null : (
                    <optgroup key={g} label={t.groups[g]}>
                      {inGroup.map((c) => (
                        <option key={c.id} value={c.id}>
                          {c.name}
                        </option>
                      ))}
                    </optgroup>
                  )
                })}
              </select>
            </label>
          </div>
        )}

        <div className="card">
          <label>
            {t.add.date}
            <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </label>
          {!editing && <p className="hint tight">{t.add.dateHint}</p>}
          <label>
            {t.add.note}
            <input value={note} maxLength={200} onChange={(e) => setNote(e.target.value)} />
          </label>
        </div>

        {error !== null && <p className="error">{error}</p>}
        <button type="submit" className="primary" disabled={busy}>
          {t.save}
        </button>
      </form>

      {editing && (
        <div className="stack tail">
          <button type="button" className={confirmDelete ? 'danger danger-on' : 'danger'} disabled={busy} onClick={() => void remove()}>
            {confirmDelete ? t.edit.confirmDelete : t.edit.delete}
          </button>
          <button type="button" className="chip" onClick={() => void toggleHistory()}>
            {events === null ? t.edit.history : t.edit.hideHistory}
          </button>
          {events !== null && (
            <div className="card">
              {events.map((ev) => (
                <EventView key={ev.id} ev={ev} currency={currency} accountName={accountName} categoryName={categoryName} />
              ))}
            </div>
          )}
        </div>
      )}
    </main>
  )
}

type EventProps = {
  ev: TxEvent
  currency: string
  accountName: Map<number, string>
  categoryName: Map<number, string>
}

function EventView({ ev, currency, accountName, categoryName }: EventProps) {
  const when = new Date(ev.at).toLocaleString('ru-RU', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
  const title = ev.action === 'create' ? t.edit.created : ev.action === 'update' ? t.edit.updated : t.edit.deleted
  const { before, after } = ev

  return (
    <div className="event">
      <div className="event-head">
        <b>{title}</b>
        <span className="hint tight">{when}</span>
      </div>
      {ev.action === 'update' && before !== null && after !== null && (
        <ul className="event-diff">
          {before.amount_minor !== after.amount_minor && (
            <li>
              {t.edit.amountChanged}: <Money minor={before.amount_minor} currency={currency} /> → <Money minor={after.amount_minor} currency={currency} />
            </li>
          )}
          {before.date !== after.date && (
            <li>
              {t.edit.dateChanged}: {before.date} → {after.date}
            </li>
          )}
          {before.note !== after.note && (
            <li>
              {t.edit.noteChanged}: «{before.note}» → «{after.note}»
            </li>
          )}
          {before.account_id !== after.account_id && (
            <li>
              {t.edit.accountChanged}: {accountName.get(before.account_id) ?? ''} → {accountName.get(after.account_id) ?? ''}
            </li>
          )}
          {before.category_id !== after.category_id && (
            <li>
              {t.edit.categoryChanged}: {categoryName.get(before.category_id ?? 0) ?? ''} → {categoryName.get(after.category_id ?? 0) ?? ''}
            </li>
          )}
        </ul>
      )}
    </div>
  )
}
