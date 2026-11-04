import { useEffect, useState } from 'react'
import { ApiError, fetchBudgetMonth, setBudgeted, type BudgetMonth, type Category } from './api'
import { suggestBudget, type Suggestion } from './budget'
import { shiftMonth } from './calendar'
import { errorText, t } from './i18n'
import { Modal } from './Modal'
import { Money } from './MoneyText'

const HISTORY = 3

type Props = {
  month: string
  // Уже заложено в этом месяце по категориям: подсказка заполняет только пустые.
  current: Map<number, number>
  categories: Category[]
  baseCurrency: string
  onApplied: (b: BudgetMonth) => void
  onClose: () => void
}

// Подсказка бюджета: средние траты по категориям за три прошлых месяца. Применяется только к категориям без суммы.
export function BudgetSuggest({ month, current, categories, baseCurrency, onApplied, onClose }: Props) {
  const [list, setList] = useState<Suggestion[] | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let alive = true
    Promise.all(Array.from({ length: HISTORY }, (_, i) => fetchBudgetMonth(shiftMonth(month, -(i + 1)))))
      .then((months) => alive && setList(suggestBudget(months)))
      .catch(() => alive && setError(t.errors.internal))
    return () => {
      alive = false
    }
  }, [month])

  const names = new Map(categories.map((c) => [c.id, c.name]))
  const empty = (list ?? []).filter((s) => (current.get(s.categoryId) ?? 0) === 0)

  async function apply() {
    setBusy(true)
    setError(null)
    try {
      let last: BudgetMonth | null = null
      for (const s of empty) last = await setBudgeted(month, s.categoryId, s.minor)
      if (last !== null) onApplied(last)
    } catch (e) {
      setError(errorText(e instanceof ApiError ? e.code : 'internal'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title={t.budget.suggest.title} onClose={onClose}>
      <p className="hint">{t.budget.suggest.hint}</p>
      {list !== null && list.length === 0 && <p className="hint">{t.budget.suggest.empty}</p>}
      <div className="list">
        {(list ?? []).map((s) => (
          <div key={s.categoryId} className="budget-top">
            <span className="entry-title">{names.get(s.categoryId) ?? ''}</span>
            <Money minor={s.minor} currency={baseCurrency} />
          </div>
        ))}
      </div>
      {error !== null && <p className="error">{error}</p>}
      {empty.length > 0 && (
        <button type="button" className="primary" disabled={busy} onClick={() => void apply()}>
          {t.budget.suggest.apply} ({empty.length})
        </button>
      )}
    </Modal>
  )
}
