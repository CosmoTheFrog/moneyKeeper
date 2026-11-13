import { t } from './i18n'
import { pressKey } from './amountKeys'

type Props = {
  value: string
  // Знаков после запятой у выбранной валюты.
  units: number
  onChange: (value: string) => void
}

const KEYS = ['1', '2', '3', '4', '5', '6', '7', '8', '9', ',', '0', 'back'] as const

// KeyPad: цифровая клавиатура для суммы вместо системной: крупные клавиши под большим пальцем, поле суммы остаётся на виду.
export function KeyPad({ value, units, onChange }: Props) {
  return (
    <div className="keypad">
      {KEYS.map((k) => {
        if (k === ',' && units === 0) return <span key={k} />
        return (
          <button key={k} type="button" className="key" aria-label={k === 'back' ? t.add.erase : k} onClick={() => onChange(pressKey(value, k, units))}>
            {k === 'back' ? (
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M9 6h11v12H9l-5-6zM12 9.5l4 4M16 9.5l-4 4" />
              </svg>
            ) : (
              k
            )}
          </button>
        )
      })}
    </div>
  )
}
