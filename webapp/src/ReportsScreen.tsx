import { useState } from 'react'
import type { Category, CategoryGroup } from './api'
import { t } from './i18n'
import type { Entry } from './ledger'
import { currencySign } from './money'
import { ChipPicker } from './Pickers'
import { PeriodPicker } from './PeriodPicker'
import { addDays, missingRates, resolveRange, spendByCategory, spendTotals, type Measure, type Period, type Range } from './report'
import { SpendBreakdown } from './SpendBreakdown'

type Props = {
  entries: Entry[]
  groups: CategoryGroup[]
  categories: Category[]
  baseCurrency: string
  today: string
}

const presets = ['week', 'twoWeeks', 'threeWeeks', 'month'] as const

const BASE = 'base'

export function ReportsScreen({ entries, groups, categories, baseCurrency, today }: Props) {
  const [period, setPeriod] = useState<Period>('month')
  const [custom, setCustom] = useState<Range>({ from: addDays(today, -29), to: today })
  // 'base': всё в базовой валюте по курсу дня каждой траты; код валюты: только траты в ней, как есть.
  const [view, setView] = useState(BASE)

  const range = resolveRange(period, custom, today, entries)
  const totals = spendTotals(entries, range, baseCurrency)
  // Пересчёт нужен, только если кроме базовой валюты что-то тратили; иначе переключать нечего.
  const mixed = totals.some((x) => x.currency !== baseCurrency)
  const chosen = mixed && totals.some((x) => x.currency === view) ? view : BASE
  const measure: Measure = chosen === BASE ? { kind: 'base' } : { kind: 'currency', code: chosen }
  const currency = chosen === BASE ? baseCurrency : chosen

  const slices = spendByCategory(entries, categories, range, measure)
  const missing = chosen === BASE ? missingRates(entries, range) : 0

  return (
    <main className="screen">
      <h1>{t.reports.title}</h1>

      <PeriodPicker
        presets={presets}
        period={period}
        range={range}
        custom={custom}
        today={today}
        onPeriod={setPeriod}
        onCustom={(r) => {
          setCustom(r)
          setPeriod('custom')
        }}
      />

      {mixed && (
        <>
          <ChipPicker
            value={chosen}
            options={[{ value: BASE, label: `${t.reports.inBase} ${currencySign(baseCurrency)}` }, ...totals.map((x) => ({ value: x.currency, label: x.currency }))]}
            onChange={setView}
          />
          <p className="hint tight">{chosen === BASE ? t.reports.inBaseHint : t.reports.ownCurrency}</p>
        </>
      )}

      <SpendBreakdown key={chosen} slices={slices} groups={groups} currency={currency} />

      {missing > 0 && (
        <p className="hint tight">
          {t.reports.missingRates}: {missing}
        </p>
      )}
    </main>
  )
}
