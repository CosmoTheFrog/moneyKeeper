import { useState } from 'react'
import type { Account, Category, CategoryGroup } from './api'
import { daysInRange } from './calendar'
import { DisplayCurrency } from './DisplayCurrency'
import { EntryList } from './EntryList'
import { t } from './i18n'
import type { Entry } from './ledger'
import { Money } from './MoneyText'
import { currencySign } from './money'
import { ChipPicker } from './Pickers'
import { PeriodPicker } from './PeriodPicker'
import { addDays, missingRates, resolveRange, spendBuckets, spendByCategory, spendOf, spendTotals, type Measure, type Period, type Range } from './report'
import { SpendBars } from './SpendBars'
import { SpendBreakdown } from './SpendBreakdown'

type Props = {
  account: Account
  // Все записи бюджета: здесь берутся только расходы этого счёта.
  entries: Entry[]
  accounts: Account[]
  groups: CategoryGroup[]
  categories: Category[]
  baseCurrency: string
  currencies: string[]
  onDisplay: (code: string) => void
  today: string
  onBack: () => void
  onOpen: (entry: Entry) => void
}

const presets = ['all', 'month', 'week'] as const

const BASE = 'base'

// Куда и когда ушли деньги со счёта: траты по группам и категориям, по дням или месяцам и списком.
// Нажатие на столбик «когда» сужает и кольцо, и список до этого дня или месяца.
export function AccountReport({ account, entries, accounts, groups, categories, baseCurrency, currencies, onDisplay, today, onBack, onOpen }: Props) {
  const [period, setPeriod] = useState<Period>('all')
  const [custom, setCustom] = useState<Range>({ from: addDays(today, -29), to: today })
  const [view, setView] = useState(BASE)
  const [bucket, setBucket] = useState<string | null>(null)

  const own = entries.filter((e) => e.kind === 'expense' && e.accountId === account.id)
  const range = resolveRange(period, custom, today, own)

  const totals = spendTotals(own, range, baseCurrency)
  const mixed = totals.some((x) => x.currency !== baseCurrency)
  const chosen = mixed && totals.some((x) => x.currency === view) ? view : BASE
  const measure: Measure = chosen === BASE ? { kind: 'base' } : { kind: 'currency', code: chosen }
  const currency = chosen === BASE ? baseCurrency : chosen

  const buckets = spendBuckets(own, range, measure)
  const picked = buckets.items.find((b) => b.key === bucket)
  const shown: Range = picked === undefined ? range : { from: picked.from, to: picked.to }

  // В списке все траты периода; при выборе одной валюты только в ней. В сумму входят те, что удалось выразить в мере.
  const listed = own.filter((e) => e.date >= shown.from && e.date <= shown.to && (chosen === BASE || e.currency === chosen))
  const total = listed.reduce((sum, e) => sum + (spendOf(e, measure) ?? 0), 0)
  const days = daysInRange(shown)
  const missing = chosen === BASE ? missingRates(own, shown) : 0

  function changePeriod(next: Period) {
    setPeriod(next)
    setBucket(null)
  }

  return (
    <main className="screen">
      <header className="row">
        <button type="button" className="chip" onClick={onBack}>
          {t.back}
        </button>
        <DisplayCurrency display={baseCurrency} currencies={currencies} onChange={onDisplay} />
      </header>

      <h1>{account.name}</h1>
      <p className="hint tight">
        {t.accountReport.title}. {t.accountReport.noteTransfers}
      </p>

      <PeriodPicker
        presets={presets}
        period={period}
        range={range}
        custom={custom}
        today={today}
        onPeriod={changePeriod}
        onCustom={(r) => {
          setCustom(r)
          changePeriod('custom')
        }}
      />

      {mixed && (
        <>
          <ChipPicker
            value={chosen}
            options={[{ value: BASE, label: `${t.reports.inBase} ${currencySign(baseCurrency)}` }, ...totals.map((x) => ({ value: x.currency, label: x.currency }))]}
            onChange={(v) => {
              setView(v)
              setBucket(null)
            }}
          />
          <p className="hint tight">{chosen === BASE ? t.reports.inBaseHint : t.reports.ownCurrency}</p>
        </>
      )}

      <div className="card stats">
        <div className="stat stat-main">
          <span className="label">{t.reports.spent}</span>
          <Money minor={total} currency={currency} />
        </div>
        <div className="stat">
          <span className="label">{t.reports.perDay}</span>
          <Money minor={Math.round(total / days)} currency={currency} />
        </div>
        <div className="stat">
          <span className="label">{t.reports.count}</span>
          <span className="stat-num">{listed.length}</span>
        </div>
      </div>
      {missing > 0 && (
        <p className="hint tight">
          {t.reports.missingRates}: {missing}
        </p>
      )}

      {own.length === 0 ? (
        <p className="hint">{t.accountReport.noSpend}</p>
      ) : (
        <>
          <h2 className="day">{t.accountReport.when}</h2>
          <SpendBars buckets={buckets} currency={currency} today={today} selected={bucket} onSelect={setBucket} />

          <h2 className="day">{t.accountReport.where}</h2>
          <SpendBreakdown key={`${chosen}-${picked?.key ?? ''}`} slices={spendByCategory(own, categories, shown, measure)} groups={groups} currency={currency} />

          <h2 className="day">
            {t.accountReport.list} · {listed.length}
          </h2>
          <EntryList entries={listed} accounts={accounts} categories={categories} today={today} onOpen={onOpen} />
        </>
      )}
    </main>
  )
}
