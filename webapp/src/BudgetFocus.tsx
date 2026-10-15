import { useState } from 'react'
import type { Account, BudgetMonth, Category } from './api'
import type { Tone } from './budget'
import { t } from './i18n'
import { Backdrop } from './Modal'
import { Money } from './MoneyText'

type Props = {
  budget: BudgetMonth
  tone: Tone
  // Счета бюджета: деньги, из которых складывается «К распределению».
  accounts: Account[]
  categories: Category[]
  baseCurrency: string
  onClose: () => void
}

type Back = 'accounts' | 'categories'

type Line = { key: number; name: string; minor: number; currency: string }

// Остаток счёта в валюте бюджета; если курса нет хотя бы у одной валюты, показывается основная валюта счёта как есть.
function accountLine(a: Account, baseCurrency: string): Line {
  const base = a.balances.map((b) => b.base_balance_minor)
  if (base.length > 0 && base.every((v) => v !== null)) {
    return { key: a.id, name: a.name, minor: base.reduce<number>((s, v) => s + (v ?? 0), 0), currency: baseCurrency }
  }
  const main = a.balances[0]
  return { key: a.id, name: a.name, minor: main?.balance_minor ?? 0, currency: main?.currency ?? a.currency }
}

// «К распределению» на переднем плане, как карточка счёта. Нажатие переворачивает её: на обороте деньги по счетам
// или по категориям, кнопка внизу переключает между ними.
export function BudgetFocus({ budget, tone, accounts, categories, baseCurrency, onClose }: Props) {
  const [flipped, setFlipped] = useState(false)
  const [back, setBack] = useState<Back>('accounts')

  const byAccount = accounts.filter((a) => !a.off_budget).map((a) => accountLine(a, baseCurrency))
  const rows = new Map(budget.categories.map((r) => [r.category_id, r]))
  const byCategory: Line[] = categories
    .map((c) => ({ key: c.id, name: c.name, minor: rows.get(c.id)?.balance_minor ?? 0, currency: baseCurrency }))
    .filter((l) => l.minor !== 0)
    .sort((a, b) => b.minor - a.minor)
  const lines = back === 'accounts' ? byAccount : byCategory

  return (
    <Backdrop onClose={onClose}>
      <div className="flip-scene">
        <div
          className={flipped ? 'flip flipped' : 'flip'}
          style={{ height: 380 }}
          role="button"
          tabIndex={0}
          aria-pressed={flipped}
          onClick={() => setFlipped(!flipped)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault()
              setFlipped(!flipped)
            }
          }}
        >
          <div className={`flip-face account account-debit focus-${tone}`}>
            <div className="row">
              <span className="account-name">{t.budget.toBudget}</span>
            </div>
            <div>
              <div className="account-balance">
                <Money minor={budget.to_budget_minor} currency={baseCurrency} />
              </div>
              <div className="account-note">{t.budget.tone[tone]}</div>
            </div>
            <div className="account-others">
              <span>
                {t.budget.income} <Money minor={budget.income_minor} currency={baseCurrency} />
              </span>
              <span>
                {t.budget.budgeted} <Money minor={budget.budgeted_minor} currency={baseCurrency} />
              </span>
            </div>
          </div>

          <div className="flip-face flip-back account account-debit">
            <div className="row">
              <span className="account-name">{back === 'accounts' ? t.budget.byAccounts : t.budget.byCategories}</span>
            </div>
            <div className="focus-list">
              {lines.length === 0 && <div className="account-note">{t.budget.noLines}</div>}
              {lines.map((l) => (
                <div key={l.key} className="spent-row">
                  <span className="focus-name">{l.name}</span>
                  <Money minor={l.minor} currency={l.currency} />
                </div>
              ))}
            </div>
            <button
              type="button"
              className="card-action"
              onClick={(e) => {
                e.stopPropagation() // кнопка не переворачивает карту
                setBack(back === 'accounts' ? 'categories' : 'accounts')
              }}
            >
              {back === 'accounts' ? t.budget.showByCategories : t.budget.showByAccounts}
            </button>
          </div>
        </div>
      </div>
    </Backdrop>
  )
}
