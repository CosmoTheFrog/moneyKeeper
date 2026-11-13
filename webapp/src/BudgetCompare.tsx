import { useEffect, useState } from 'react'
import { scaleMonth } from './displayFactor'
import { fetchBudgetMonth, type BudgetMonth, type Category } from './api'
import { compareMonths } from './budget'
import { monthTitle, shiftMonth } from './calendar'
import { t } from './i18n'
import { Modal } from './Modal'
import { Money } from './MoneyText'

type Props = {
  month: string
  budget: BudgetMonth
  categories: Category[]
  baseCurrency: string
  // Курс пересчёта месяца из базовой валюты в показываемую; budget уже пересчитан.
  factor: number
  // Показывать заложенные суммы: нужны только в режиме конвертов.
  plan: boolean
  onClose: () => void
}

// Сравнение месяца с другим: по каждой категории план и трата в обоих месяцах и что изменилось.
export function BudgetCompare({ month, budget, categories, baseCurrency, factor, plan, onClose }: Props) {
  const [other, setOther] = useState(shiftMonth(month, -1))
  const [then, setThen] = useState<BudgetMonth | null>(null)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let alive = true
    setFailed(false)
    fetchBudgetMonth(other)
      .then((b) => alive && setThen(scaleMonth(b, factor)))
      .catch(() => alive && setFailed(true))
    return () => {
      alive = false
    }
  }, [other, factor])

  const names = new Map(categories.map((c) => [c.id, c.name]))
  const rows = then === null ? [] : compareMonths(budget, then)

  return (
    <Modal title={t.budget.compare.title} onClose={onClose}>
      <div className="month-nav">
        <button type="button" className="plus" aria-label={t.calendar.prev} onClick={() => setOther(shiftMonth(other, -1))}>
          ‹
        </button>
        <span className="month-title">
          {monthTitle(month)} {t.budget.compare.vs} {monthTitle(other)}
        </span>
        <button type="button" className="plus" aria-label={t.calendar.next} disabled={other >= month} onClick={() => setOther(shiftMonth(other, 1))}>
          ›
        </button>
      </div>
      {failed && <p className="error">{t.errors.internal}</p>}
      {then !== null && rows.length === 0 && <p className="hint">{t.budget.compare.empty}</p>}
      <div className="list">
        {rows.map((r) => (
          <div key={r.categoryId} className="budget-row">
            <div className="budget-top">
              <span className="entry-title">{names.get(r.categoryId) ?? ''}</span>
              <span className="budget-balance">
                <Money minor={r.spentDelta} currency={baseCurrency} signed />
                <span className="budget-cap">{r.spentDelta > 0 ? t.budget.compare.more : r.spentDelta < 0 ? t.budget.compare.less : t.budget.compare.same}</span>
              </span>
            </div>
            <div className="budget-nums">
              <span>
                {t.budget.spent} <Money minor={r.spentNow} currency={baseCurrency} /> / <Money minor={r.spentThen} currency={baseCurrency} />
              </span>
              {plan && (
                <span>
                  {t.budget.planned}
                  <Money minor={r.budgetedNow} currency={baseCurrency} /> / <Money minor={r.budgetedThen} currency={baseCurrency} />
                </span>
              )}
            </div>
          </div>
        ))}
      </div>
    </Modal>
  )
}
