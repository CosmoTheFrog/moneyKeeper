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
import { allows, pickAccount, usableCurrencies } from './accounts'
import { errorText, t } from './i18n'
import type { Entry } from './ledger'
import { Modal } from './Modal'
import { Money } from './MoneyText'
import { formatInput, formatMinor, parseAmount } from './money'
import { ChipPicker, CurrencyPicker, SheetPicker } from './Pickers'

type FormKind = 'expense' | 'income' | 'transfer' | 'opening'

type Props = {
  accounts: Account[]
  groups: CategoryGroup[]
  categories: Category[]
  // Без entry форма создаёт запись, с entry правит её.
  entry?: Entry
  // Все валюты, которые знает приложение; на многовалютном счёте доступна любая из них.
  allCurrencies: string[]
  // Категория, которую нужно выбрать заранее (кнопка «+» у категории).
  presetCategoryId?: number
  onDone: () => void
  onCancel: () => void
}

const kindOptions = (['expense', 'income', 'transfer'] as const).map((k) => ({ value: k, label: t.kinds[k] }))

function kindLabel(kind: FormKind): string {
  return kind === 'opening' ? t.ledger.opening : t.kinds[kind]
}

export function EntryForm({ accounts, groups, categories, entry, allCurrencies, presetCategoryId, onDone, onCancel }: Props) {
  const editing = entry !== undefined
  const defaultAccount = accounts.find((a) => a.is_default) ?? accounts[0]

  const [kind, setKind] = useState<FormKind>(entry?.kind ?? 'expense')
  const [quick, setQuick] = useState('')
  const [understood, setUnderstood] = useState<string | null>(null)
  const [amount, setAmount] = useState(entry === undefined ? '' : formatInput(entry.amountMinor, entry.currency))
  const [currency, setCurrency] = useState(entry?.currency ?? defaultAccount?.currency ?? '')
  const [accountId, setAccountId] = useState(entry?.accountId ?? defaultAccount?.id ?? 0)
  const [toAccountId, setToAccountId] = useState(entry?.toAccountId ?? 0)
  const [categoryId, setCategoryId] = useState(entry?.categoryId ?? presetCategoryId ?? 0)
  const [date, setDate] = useState(entry?.date ?? '')
  const [note, setNote] = useState(entry?.note ?? '')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [events, setEvents] = useState<TxEvent[] | null>(null)

  const currencies = usableCurrencies(accounts, allCurrencies)
  const accountsInCurrency = accounts.filter((a) => allows(a, currency))
  const accountName = new Map(accounts.map((a) => [a.id, a.name]))
  const categoryName = new Map(categories.map((c) => [c.id, c.name]))

  const canPickAccount = editing ? kind === 'expense' || kind === 'income' : true
  const canPickTarget = !editing && kind === 'transfer'
  const targets = accountsInCurrency.filter((a) => a.id !== accountId)

  const categoryGroups = groups
    .map((g) => ({
      title: g.name,
      options: categories.filter((c) => c.group_id === g.id &&(!c.hidden || c.id === entry?.categoryId)).map((c) => ({ value: c.id, label: c.name })),
    }))
    .filter((g) => g.options.length > 0)

  // Смена валюты оставляет счёт, если он её допускает (многовалютный), иначе переключает на основной счёт этой валюты.
  // Если подходящего счёта нет, счёт сбрасывается.
  function chooseCurrency(next: string) {
    setCurrency(next)
    setAccountId(pickAccount(accounts, next, accountId))
    setToAccountId(0)
  }

  function selectAccount(id: number) {
    setAccountId(id)
    if (id === toAccountId) setToAccountId(0)
  }

  async function understand() {
    const text = quick.trim()
    if (text === '') return
    setError(null)
    try {
      const p = await parseQuick(text, currency)
      setAmount(formatInput(p.amount_minor, p.currency !== '' ? p.currency : currency))
      if (p.note !== '') setNote(p.note)
      let cur = currency
      if (p.currency !== '' && p.currency !== currency) {
        if (currencies.includes(p.currency)) chooseCurrency(p.currency)
        else setError(t.add.noAccountInCurrency)
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
    const minor = parseAmount(amount, currency)
    if (minor === null) return setError(errorText('invalid_amount'))
    if (accountId === 0) return setError(t.add.noAccountInCurrency)
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
          currency,
          date: date === '' ? undefined : date,
          note,
        })
      } else {
        const patch: TransactionPatch = {}
        if (minor !== Math.abs(entry.amountMinor)) patch.amount_minor = minor
        if (date !== entry.date) patch.date = date
        if (note !== entry.note) patch.note = note
        if (canPickAccount && accountId !== entry.accountId) patch.account_id = accountId
        if (canPickAccount && currency !== entry.currency) patch.currency = currency
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
    <Modal title={editing ? t.edit.title : t.add.title} onClose={onCancel}>
      {!editing && (
        <div className="top-gap">
          <ChipPicker value={kind === 'opening' ? 'expense' : kind} options={kindOptions} onChange={setKind} />
        </div>
      )}
      {editing && <p className="hint">{kindLabel(kind)}</p>}

      <form className="stack" onSubmit={submit}>
        {!editing && (
          <div className="card">
            <label className="field">
              <span className="label">{t.add.quick}</span>
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
          <label className="field">
            <span className="label">{t.add.amount}</span>
            <span className="amount-row">
              <input inputMode="decimal" value={amount} placeholder="0,00" onChange={(e) => setAmount(e.target.value)} />
            </span>
          </label>
          {currencies.length > 0 && (
            <CurrencyPicker label={t.add.currency} value={currency} options={currencies} onChange={chooseCurrency} disabled={editing && !canPickAccount} />
          )}
          {editing && !canPickAccount && <p className="hint tight">{t.edit.currencyLocked}</p>}
        </div>

        {canPickAccount && (
          <div className="card">
            <span className="label">{canPickTarget ? t.add.fromAccount : t.add.account}</span>
            <div className="choices">
              {accountsInCurrency.map((a) => (
                <button key={a.id} type="button" className={a.id === accountId ? 'choice choice-on' : 'choice'} onClick={() => selectAccount(a.id)}>
                  <span>{a.name}</span>
                  <span className="choice-meta">{a.currency}</span>
                </button>
              ))}
              {accountsInCurrency.length === 0 && <p className="hint tight">{t.add.noAccountInCurrency}</p>}
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
            <SheetPicker label={t.add.category} value={categoryId === 0 ? null : categoryId} placeholder={t.add.pickCategory} groups={categoryGroups} onChange={setCategoryId} />
          </div>
        )}

        <div className="card">
          <label className="field">
            <span className="label">{t.add.date}</span>
            <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </label>
          {!editing && <p className="hint tight">{t.add.dateHint}</p>}
          <label className="field">
            <span className="label">{t.add.note}</span>
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
    </Modal>
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
