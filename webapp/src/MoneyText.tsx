import { createContext, useContext, type ReactNode } from 'react'
import { t } from './i18n'
import { currencySign, formatMinor } from './money'

type Privacy = {
  hidden: boolean
  toggle: () => void
  // false: сумма не нажимается (экраны без режима скрытия).
  switchable?: boolean
}

export const PrivacyContext = createContext<Privacy>({ hidden: false, toggle: () => {} })

// NoPrivacy: экраны, где суммы всегда видны (бюджет, отчёты, накопления, портфель, платежи). Режим скрытия сумм
// работает только на вкладках «Операции» и «Счета», поэтому здесь сумма это просто текст.
export function NoPrivacy({ children }: { children: ReactNode }) {
  return <PrivacyContext.Provider value={{ hidden: false, toggle: () => {}, switchable: false }}>{children}</PrivacyContext.Provider>
}

type Props = {
  minor: number
  currency: string
  signed?: boolean
}

// Все суммы в интерфейсе показываются только этим компонентом: табличные цифры и звёздочки при режиме приватности.
// Нажатие на цифры переключает режим для всех сумм сразу.
export function Money({ minor, currency, signed = false }: Props) {
  const { hidden, toggle, switchable = true } = useContext(PrivacyContext)
  const tone = minor < 0 ? 'neg' : minor > 0 && signed ? 'pos' : ''
  if (!switchable) return <span className={`money ${tone}`}>{formatMinor(minor, currency, signed)}</span>
  return (
    <button
      type="button"
      className={`money ${tone}`}
      aria-label={hidden ? t.privacy.show : t.privacy.hide}
      onClick={(e) => {
        e.stopPropagation() // сумма лежит внутри нажимаемых строк и карточек, они на неё не реагируют
        toggle()
      }}
    >
      {hidden ? `*** ${currencySign(currency)}` : formatMinor(minor, currency, signed)}
    </button>
  )
}
