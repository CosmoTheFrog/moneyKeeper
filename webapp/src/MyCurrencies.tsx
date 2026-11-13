import { t } from './i18n'
import { Modal } from './Modal'
import { currencySign, isCoin, isStock } from './money'

type Props = {
  registry: string[]
  enabled: string[]
  // Используются базой, показом или счетами: выключить нельзя.
  pinned: string[]
  onChange: (list: string[]) => void
  onClose: () => void
}

// Мои валюты: какие валюты предлагать в формах и курсах. Выбор хранится на устройстве.
export function MyCurrencies({ registry, enabled, pinned, onChange, onClose }: Props) {
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
