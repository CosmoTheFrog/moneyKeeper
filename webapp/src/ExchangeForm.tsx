import { useState, type FormEvent } from 'react'
import { ApiError, createExchange, type Account, type Category, type CategoryGroup } from './api'
import { errorText, t } from './i18n'
import { currencySign, dealPrice, formatRate, isCoin, parseAmount } from './money'
import { CurrencyPicker, SheetPicker } from './Pickers'

type Props = {
  accounts: Account[]
  groups: CategoryGroup[]
  categories: Category[]
  allCurrencies: string[]
  onDone: () => void
}

// currenciesOf: на многовалютном счёте любая валюта, на обычном только его.
function currenciesOf(a: Account | undefined, all: string[]): string[] {
  if (a === undefined) return []
  return a.multi_currency ? all : [a.currency]
}

// Подпись цены сделки. Если отдают монету за обычную валюту, это цена целой монеты, иначе цена целой единицы получаемой валюты.
function dealLine(out: number, outCur: string, inn: number, inCur: string): string | null {
  const coinGiven = isCoin(outCur) && !isCoin(inCur)
  const [paid, got] = coinGiven ? [inCur, outCur] : [outCur, inCur]
  const price = coinGiven ? dealPrice(inn, inCur, out, outCur) : dealPrice(out, outCur, inn, inCur)
  if (price === '') return null
  return `${t.exchange.deal}: ${formatRate(price)} ${currencySign(paid)} ${t.exchange.per} ${currencySign(got)}`
}
// Обмен: отдаю одну сумму, получаю другую, обе как провёл банк или биржа; курс считается сам. Через границу бюджета
// нужна категория, комиссия со счёта внутри бюджета уходит расходом в свою категорию.
export function ExchangeFields({ accounts, groups, categories, allCurrencies, onDone }: Props) {
  const first = accounts.find((a) => a.is_default) ?? accounts[0]
  const [fromId, setFromId] = useState(first?.id ?? 0)
  const [toId, setToId] = useState(0)
  const [outCur, setOutCur] = useState(first?.currency ?? '')
  const [inCur, setInCur] = useState('')
  const [outAmount, setOutAmount] = useState('')
  const [inAmount, setInAmount] = useState('')
  const [feeAmount, setFeeAmount] = useState('')
  const [categoryId, setCategoryId] = useState(0)
  const [feeCategoryId, setFeeCategoryId] = useState(0)
  const [date, setDate] = useState('')
  const [note, setNote] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const from = accounts.find((a) => a.id === fromId)
  const to = accounts.find((a) => a.id === toId)
  const crossing = from !== undefined && to !== undefined && !from.off_budget && to.off_budget
  const feeOnBudget = from !== undefined && !from.off_budget

  const categoryGroups = groups
    .map((g) => ({
      title: g.name,
      options: categories.filter((c) => c.group_id === g.id && !c.hidden).map((c) => ({ value: c.id, label: c.name })),
    }))
    .filter((g) => g.options.length > 0)

  function pickFrom(a: Account) {
    setFromId(a.id)
    setOutCur(a.multi_currency && outCur !== '' ? outCur : a.currency)
  }

  function pickTo(a: Account) {
    setToId(a.id)
    // Копилка сама знает свой конверт.
    if (a.kind === 'piggy' && a.category_id !== null) setCategoryId(a.category_id)
    setInCur(a.multi_currency && inCur !== '' && inCur !== outCur ? inCur : a.currency)
  }

  const outMinor = parseAmount(outAmount, outCur)
  const inMinor = parseAmount(inAmount, inCur)
  const deal = outMinor !== null && inMinor !== null ? dealLine(outMinor, outCur, inMinor, inCur) : null

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    if (from === undefined || to === undefined) return setError(t.exchange.pickAccounts)
    if (outMinor === null || inMinor === null) return setError(errorText('invalid_amount'))
    if (outCur === inCur) return setError(t.exchange.sameCurrency)
    const fee = feeAmount.trim() === '' ? 0 : parseAmount(feeAmount, outCur)
    if (fee === null) return setError(errorText('invalid_amount'))
    if (crossing && categoryId === 0) return setError(t.add.pickCategory)
    if (fee > 0 && feeOnBudget && feeCategoryId === 0) return setError(t.add.pickCategory)

    setBusy(true)
    try {
      await createExchange({
        from_account_id: from.id,
        to_account_id: to.id,
        out_minor: outMinor,
        out_currency: outCur,
        in_minor: inMinor,
        in_currency: inCur,
        category_id: crossing ? categoryId : undefined,
        fee_minor: fee > 0 ? fee : undefined,
        fee_category_id: fee > 0 && feeOnBudget ? feeCategoryId : undefined,
        date: date === '' ? undefined : date,
        note,
      })
      onDone()
    } catch (err) {
      setError(errorText(err instanceof ApiError ? err.code : 'internal'))
      setBusy(false)
    }
  }

  if (accounts.length === 0) return <p className="hint">{t.exchange.noAccounts}</p>

  return (
    <form className="stack" onSubmit={submit}>
      <div className="card">
        <span className="label">{t.exchange.give}</span>
        <div className="choices">
          {accounts.map((a) => (
            <button key={a.id} type="button" className={a.id === fromId ? 'choice choice-on' : 'choice'} onClick={() => pickFrom(a)}>
              <span>{a.name}</span>
              <span className="choice-meta">{a.multi_currency ? t.accounts.many : a.currency}</span>
            </button>
          ))}
        </div>
        <label className="field">
          <span className="label">{t.exchange.amount}</span>
          <input inputMode="decimal" value={outAmount} placeholder="0,00" onChange={(e) => setOutAmount(e.target.value)} />
        </label>
        {from?.multi_currency === true && <CurrencyPicker value={outCur} options={currenciesOf(from, allCurrencies)} onChange={setOutCur} />}
      </div>

      <div className="card">
        <span className="label">{t.exchange.get}</span>
        <div className="choices">
          {accounts.map((a) => (
            <button key={a.id} type="button" className={a.id === toId ? 'choice choice-on' : 'choice'} onClick={() => pickTo(a)}>
              <span>{a.name}</span>
              <span className="choice-meta">{a.multi_currency ? t.accounts.many : a.currency}</span>
            </button>
          ))}
        </div>
        <label className="field">
          <span className="label">{t.exchange.amount}</span>
          <input inputMode="decimal" value={inAmount} placeholder="0,00" onChange={(e) => setInAmount(e.target.value)} />
        </label>
        {to?.multi_currency === true && <CurrencyPicker value={inCur} options={currenciesOf(to, allCurrencies)} onChange={setInCur} />}
      </div>

      {deal !== null && (
        <div className="card deal">
          <b>{deal}</b>
          <p className="hint tight">{t.exchange.dealHint}</p>
        </div>
      )}

      {from !== undefined && to !== undefined && (crossing || (from.off_budget && to.off_budget)) && (
        <div className="card">
          {crossing ? (
            <>
              <p className="hint tight">{t.exchange.leavesBudget}</p>
              <SheetPicker label={t.add.category} value={categoryId === 0 ? null : categoryId} placeholder={t.add.pickCategory} groups={categoryGroups} onChange={setCategoryId} />
            </>
          ) : (
            from.off_budget && to.off_budget && <p className="hint tight">{t.exchange.insideHint}</p>
          )}
        </div>
      )}

      <div className="card">
        <label className="field">
          <span className="label">{t.exchange.fee}</span>
          <input inputMode="decimal" value={feeAmount} placeholder="0,00" onChange={(e) => setFeeAmount(e.target.value)} />
        </label>
        {feeAmount.trim() !== '' && (
          <>
            <p className="hint tight">{feeOnBudget ? t.exchange.feeBudgetHint : t.exchange.feeWalletHint}</p>
            {feeOnBudget && (
              <SheetPicker label={t.exchange.feeCategory} value={feeCategoryId === 0 ? null : feeCategoryId} placeholder={t.add.pickCategory} groups={categoryGroups} onChange={setFeeCategoryId} />
            )}
          </>
        )}
      </div>

      <div className="card">
        <label className="field">
          <span className="label">{t.add.date}</span>
          <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </label>
        <p className="hint tight">{t.add.dateHint}</p>
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
  )
}
