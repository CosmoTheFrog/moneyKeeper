import { t } from './i18n'
import { Money } from './MoneyText'

type Props = {
  currentMinor: number
  previousMinor: number
  currency: string
}

// ReportCompare: синяя плитка «этот период против прошлого» двумя полосками. Не показывается, если раньше трат не было.
export function ReportCompare({ currentMinor, previousMinor, currency }: Props) {
  if (previousMinor <= 0) return null
  const top = Math.max(currentMinor, previousMinor)
  const width = (v: number) => `${Math.max((v / top) * 100, 2)}%`
  return (
    <div className="bt bt-compare">
      <span className="bt-cap">{t.reports.compare}</span>
      <div className="cmp-row">
        <span>{t.reports.thisPeriod}</span>
        <i>
          <b style={{ width: width(currentMinor) }} />
        </i>
        <Money minor={currentMinor} currency={currency} />
      </div>
      <div className="cmp-row cmp-prev">
        <span>{t.reports.prevPeriod}</span>
        <i>
          <b style={{ width: width(previousMinor) }} />
        </i>
        <Money minor={previousMinor} currency={currency} />
      </div>
    </div>
  )
}
