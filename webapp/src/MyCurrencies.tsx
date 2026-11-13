import { useState, type FormEvent } from 'react'
import { addCurrency, ApiError } from './api'
import { errorText, t } from './i18n'
import { Modal } from './Modal'
import { currencySign, isCoin, isStock, registerCurrency } from './money'

type Props = {
  registry: string[]
  enabled: string[]
  // Используются базой, показом или счетами: выключить нельзя.
  pinned: string[]
  onChange: (list: string[]) => void
  // Добавлена новая валюта: списки в приложении нужно перечитать.
  onAdded: () => void
  onClose: () => void
}

// Мои валюты: какие валюты предлагать в формах и курсах. Выбор хранится на устройстве.
export function MyCurrencies({ registry, enabled, pinned, onChange, onAdded, onClose }: Props) {
  const [code, setCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function add(e: FormEvent) {
    e.preventDefault()
    if (code.trim() === '') return
    setError(null)
    setBusy(true)
    try {
      const info = await addCurrency(code.trim())
      registerCurrency(info)
      if (!enabled.includes(info.code)) onChange([...enabled, info.code])
      setCode('')
      onAdded()
    } catch (err) {
      const apiCode = err instanceof ApiError ? err.code : 'internal'
      setError(apiCode === 'asset_not_found' ? t.myCurrencies.notFound : errorText(apiCode))
    }
    setBusy(false)
  }

  function toggle(code: string) {
    onChange(enabled.includes(code) ? enabled.filter((c) => c !== code) : [...enabled, code])
  }
  const groups = [
    { title: t.myCurrencies.plain, list: registry.filter((c) => !isCoin(c) && !isStock(c)) },
    { title: t.myCurrencies.coins, list: registry.filter((c) => isCoin(c) && !isStock(c)) },
    { title: t.myCurrencies.stocks, list: registry.filter(isStock) },
  ].filter((g) => g.list.length > 0)

  return (
    <Modal title={t.myCurrencies.title} sheet onClose={onClose}>
      <p className="hint tight">{t.myCurrencies.hint}</p>
      <form className="card" onSubmit={(e) => void add(e)}>
        <label className="field">
          <span className="label">{t.myCurrencies.add}</span>
          <input value={code} maxLength={3} autoCapitalize="characters" placeholder={t.myCurrencies.codePlaceholder} onChange={(e) => setCode(e.target.value.toUpperCase())} />
        </label>
        <p className="hint tight">{t.myCurrencies.addHint}</p>
        {error !== null && <p className="error">{error}</p>}
        <button type="submit" className="primary" disabled={busy || code.trim().length !== 3}>
          {t.myCurrencies.addButton}
        </button>
      </form>
      {groups.map((g) => (
        <div key={g.title} className="field">
          <span className="label">{g.title}</span>
          {g.list.map((c) => {
            const locked = pinned.includes(c)
            return (
              <label key={c} className="switch">
                <input type="checkbox" checked={locked || enabled.includes(c)} disabled={locked} onChange={() => toggle(c)} />
                <span className="switch-track" aria-hidden="true" />
                <span>
                  {currencySign(c)} {c !== currencySign(c) ? c : ''}
                  {locked && <small> · {t.myCurrencies.locked}</small>}
                </span>
              </label>
            )
          })}
        </div>
      ))}
    </Modal>
  )
}
