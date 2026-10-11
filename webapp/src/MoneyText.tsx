import { createContext, useContext } from 'react'
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
export function Money({ minor, currency, signed = false }: Props) {
  const { hidden } = useContext(PrivacyContext)
  const tone = minor < 0 ? 'neg' : minor > 0 && signed ? 'pos' : ''
  return <span className={`money ${tone}`}>{hidden ? `*** ${currencySign(currency)}` : formatMinor(minor, currency, signed)}</span>
}
