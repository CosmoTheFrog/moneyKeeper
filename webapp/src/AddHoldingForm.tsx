import { useState, type FormEvent } from 'react'
import { addAsset, ApiError, createAccount, createTransaction, type Account } from './api'
import { DateField } from './DateField'
import { errorText, t } from './i18n'
import { isCoin, parseAmount, registerCurrency } from './money'
import { ChipPicker, CurrencyPicker, Segmented } from './Pickers'

type Props = {
  wallets: Account[]
  allCurrencies: string[]
  today: string
  onDone: () => void
  // Добавлен новый актив: списки валют в приложении нужно обновить.
  onAssetAdded: () => void
}

const NEW_WALLET = 0

// Добавить в портфель то, что уже есть: монеты или акции, сколько штук, в каком кошельке и с какого дня.
// Это поступление в кошелёк; цена покупки берётся по рыночному курсу выбранного дня.
export function AddHoldingForm({ wallets, allCurrencies, today, onDone, onAssetAdded }: Props) {
  // Активы, добавленные в этой форме: список из приложения обновится только после перезагрузки данных.
  const [added, setAdded] = useState<string[]>([])
  const assets = [...allCurrencies.filter(isCoin), ...added.filter((c) => !allCurrencies.includes(c))]
  const [asset, setAsset] = useState(assets[0] ?? '')
  const [quantity, setQuantity] = useState('')
  const [walletId, setWalletId] = useState(wallets[0]?.id ?? NEW_WALLET)
  const [walletName, setWalletName] = useState('')
  const [date, setDate] = useState(today)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [finding, setFinding] = useState(false)
  const [code, setCode] = useState('')
  const [assetKind, setAssetKind] = useState<'coin' | 'stock'>('coin')

  async function find() {
    setError(null)
    setBusy(true)
    try {
      const info = await addAsset(code.trim(), assetKind === 'stock')
      registerCurrency(info)
      setAdded((list) => [...list, info.code])
      setAsset(info.code)
      setFinding(false)
      setCode('')
      onAssetAdded()
    } catch (err) {
      setError(errorText(err instanceof ApiError ? err.code : 'internal'))
    }
    setBusy(false)
  }

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
      <button type="button" className="link-quiet" onClick={() => setFinding(!finding)}>
        {t.portfolioAdd.notListed}
      </button>
      {finding && (
        <div className="card find-asset">
          <Segmented value={assetKind} options={[{ value: 'coin' as const, label: t.portfolioAdd.kindCoin }, { value: 'stock' as const, label: t.portfolioAdd.kindStock }]} onChange={setAssetKind} />
          <label className="field">
            <span className="label">{t.portfolioAdd.code}</span>
            <input value={code} maxLength={10} autoCapitalize="characters" placeholder={t.portfolioAdd.codePlaceholder} onChange={(e) => setCode(e.target.value.toUpperCase())} />
          </label>
          <p className="hint tight">{t.portfolioAdd.findHint}</p>
          <button type="button" className="primary" disabled={busy || code.trim() === ''} onClick={() => void find()}>
            {t.portfolioAdd.find}
          </button>
        </div>
      )}
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
