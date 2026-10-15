import { useState, type FormEvent } from 'react'
import { ApiError, createAccount, createTransaction, type Account } from './api'
import { DateField } from './DateField'
import { errorText, t } from './i18n'
import { isCoin, parseAmount } from './money'
import { ChipPicker, CurrencyPicker } from './Pickers'

type Props = {
  wallets: Account[]
  allCurrencies: string[]
  today: string
  onDone: () => void
}

const NEW_WALLET = 0

// Добавить в портфель то, что уже есть: монеты или акции, сколько штук, в каком кошельке и с какого дня.
// Это поступление в кошелёк; цена покупки берётся по рыночному курсу выбранного дня.
export function AddHoldingForm({ wallets, allCurrencies, today, onDone }: Props) {
  const assets = allCurrencies.filter(isCoin)
  const [asset, setAsset] = useState(assets[0] ?? '')
  const [quantity, setQuantity] = useState('')
  const [walletId, setWalletId] = useState(wallets[0]?.id ?? NEW_WALLET)
  const [walletName, setWalletName] = useState('')
  const [date, setDate] = useState(today)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    const minor = parseAmount(quantity, asset)
    if (minor === null) return setError(errorText('invalid_amount'))
    setBusy(true)
    try {
      const id = walletId === NEW_WALLET ? (await createAccount(walletName.trim(), 'wallet', asset, true, 0)).id : walletId
      await createTransaction({ kind: 'income', account_id: id, amount_minor: minor, currency: asset, date, note: '' })
      onDone()
    } catch (err) {
      setError(errorText(err instanceof ApiError ? err.code : 'internal'))
      setBusy(false)
    }
  }

  const walletOptions = [...wallets.map((w) => ({ value: w.id, label: w.name })), { value: NEW_WALLET, label: t.portfolioAdd.newWallet }]

  return (
    <form className="form" onSubmit={submit}>
      <CurrencyPicker label={t.portfolioAdd.asset} value={asset} options={assets} onChange={setAsset} />
      <label className="field">
        <span className="label">{t.portfolioAdd.quantity}</span>
        <input inputMode="decimal" value={quantity} placeholder="0" autoFocus onChange={(e) => setQuantity(e.target.value)} />
      </label>
      <ChipPicker label={t.portfolioAdd.wallet} value={walletId} options={walletOptions} onChange={setWalletId} />
      {walletId === NEW_WALLET && (
        <label className="field">
          <span className="label">{t.portfolioAdd.walletName}</span>
          <input value={walletName} maxLength={40} placeholder={t.portfolioAdd.walletPlaceholder} onChange={(e) => setWalletName(e.target.value)} />
        </label>
      )}
      <DateField label={t.portfolioAdd.date} value={date} onChange={setDate} placeholder={t.add.dateToday} clearable={false} />
      <p className="hint tight">{t.portfolioAdd.hint}</p>
      {error !== null && <p className="error">{error}</p>}
      <button type="submit" className="primary" disabled={busy || (walletId === NEW_WALLET && walletName.trim() === '')}>
        {t.portfolioAdd.submit}
      </button>
    </form>
  )
}
