import { createContext, useContext } from 'react'
import { t } from './i18n'
import { currencySign, formatMinor } from './money'

type Privacy = {
  hidden: boolean
  toggle: () => void
}

export const PrivacyContext = createContext<Privacy>({ hidden: false, toggle: () => {} })

type Props = {
  minor: number
  currency: string
  signed?: boolean
}

// Все суммы в интерфейсе показываются только этим компонентом: табличные цифры и звёздочки при режиме приватности.
// Нажатие на цифры переключает режим для всех сумм сразу.
export function Money({ minor, currency, signed = false }: Props) {
  const { hidden, toggle } = useContext(PrivacyContext)
  const tone = minor < 0 ? 'neg' : minor > 0 && signed ? 'pos' : ''
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
