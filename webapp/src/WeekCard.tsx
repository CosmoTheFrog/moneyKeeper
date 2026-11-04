import type { Category } from './api'
import { t } from './i18n'
import type { Entry } from './ledger'
import { Money } from './MoneyText'
import { weekSummary } from './weekSummary'

type Props = {
  entries: Entry[]
  categories: Category[]
  // Валюта, в которой записи пересчитаны (baseMinor).
  currency: string
  today: string
}

// Итог последних семи дней: сколько потрачено, что изменилось к неделе раньше и на что ушло больше всего.
export function WeekCard({ entries, categories, currency, today }: Props) {
  const s = weekSummary(entries, categories, today)
  if (s === null) return null
  const delta = s.totalMinor - s.previousMinor
  return (
    <div className="card week-card">
      <span className="label">{t.week.title}</span>
      <Money minor={s.totalMinor} currency={currency} />
      <p className="hint tight">
        {delta === 0 ? (
          t.week.same
        ) : (
          <>
            {t.week.by} <Money minor={Math.abs(delta)} currency={currency} /> {delta > 0 ? t.week.more : t.week.less}
          </>
        )}
      </p>
      {s.top !== null && (
        <p className="hint tight">
          {t.week.top}: {s.top.label} <Money minor={s.top.totalMinor} currency={currency} />
        </p>
      )}
      {s.missingRates > 0 && (
        <p className="hint tight">
          {t.reports.missingRates}: {s.missingRates}
        </p>
      )}
    </div>
  )
}
