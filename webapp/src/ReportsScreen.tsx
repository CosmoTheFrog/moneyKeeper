import { useState } from 'react'
import type { Account, Category, CategoryGroup } from './api'
import { DisplayCurrency } from './DisplayCurrency'
import { t } from './i18n'
import type { Entry } from './ledger'
import { currencySign } from './money'
import { ChipPicker } from './Pickers'
import { PeriodPicker } from './PeriodPicker'
import { missingRates, rangeFor, spendByCategory, spendTotals, type Measure, type Period } from './report'
import { SpendBreakdown } from './SpendBreakdown'

type Props = {
  entries: Entry[]
  accounts: Account[]
  groups: CategoryGroup[]
  categories: Category[]
  // Валюта, в которой посчитаны суммы «по курсу», и список валют для её выбора.
  baseCurrency: string
  currencies: string[]
  onDisplay: (code: string) => void
  today: string
  onOpen: (entry: Entry) => void
}

const BASE = 'base'

export function ReportsScreen({ entries, accounts, onOpen, groups, categories, baseCurrency, currencies, onDisplay, today }: Props) {
  const [period, setPeriod] = useState<Period>('month')
  const [offset, setOffset] = useState(0)
  // 'base': всё в базовой валюте по курсу дня каждой траты; код валюты: только траты в ней, как есть.
  const [view, setView] = useState(BASE)

  const range = rangeFor(period, today, offset)
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
      <header className="row">
        <h1>{t.reports.title}</h1>
        <DisplayCurrency display={baseCurrency} currencies={currencies} onChange={onDisplay} />
      </header>

      <PeriodPicker
        period={period}
        offset={offset}
        range={range}
        today={today}
        onPeriod={(p) => {
          setPeriod(p)
          setOffset(0)
        }}
        onOffset={setOffset}
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

      <SpendBreakdown
        key={chosen}
        slices={slices}
        groups={groups}
        categories={categories}
        accounts={accounts}
        entries={entries}
        range={range}
        measure={measure}
        currency={currency}
        today={today}
        onOpen={onOpen}
      />

      {missing > 0 && (
        <p className="hint tight">
          {t.reports.missingRates}: {missing}
        </p>
      )}
    </main>
  )
}
