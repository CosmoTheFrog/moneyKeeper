import { useState } from 'react'
import type { Account } from './api'
import { t } from './i18n'
import { Backdrop } from './Modal'
import { Money } from './MoneyText'
import { percent } from './report'

type Props = {
  account: Account
  baseCurrency: string
  // Открыть отчёт «куда ушли деньги» по этому счёту.
  onReport: () => void
  onClose: () => void
}

// Карточка счёта на переднем плане: всё остальное размыто, закрывается нажатием на фон. Нажатие переворачивает её.
// На обороте расходы по этому счёту за всё время, для счёта с несколькими валютами отдельно по каждой.
export function AccountFocus({ account, baseCurrency, onReport, onClose }: Props) {
  const [flipped, setFlipped] = useState(false)
  const [main, ...others] = account.balances
  const spending = account.balances.filter((b) => b.spent_minor > 0)

  // Итог в базовой валюте по курсу дня каждой траты и доли валют в нём: только если курс есть у всех строк.
  const baseSpent = spending.map((b) => b.base_spent_minor)
  const known = baseSpent.every((v) => v !== null)
  const baseTotal = known ? baseSpent.reduce<number>((a, b) => a + (b ?? 0), 0) : null
  const converted = spending.length > 1 || spending.some((b) => b.currency !== baseCurrency)
  const shares = spending.length > 1 && baseTotal !== null && baseTotal > 0

  // Карточка растёт вместе с числом валют: у обычного счёта это 230, у многовалютного больше.
  const rows = Math.max(account.balances.length, spending.length, 1)
  const height = Math.max(230, 196 + 34 * rows)

  return (
    <Backdrop onClose={onClose}>
      <div className="flip-scene">
        <div
          className={flipped ? 'flip flipped' : 'flip'}
          style={{ height }}
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
          <div className={`flip-face account account-${account.kind}`}>
            <div className="row">
              <span className="account-name">{account.name}</span>
              {account.is_default && <span className="badge">{t.accounts.defaultBadge}</span>}
            </div>
            <div>
              {main !== undefined && (
                <div className="account-balance">
                  <Money minor={main.balance_minor} currency={main.currency} />
                </div>
              )}
              {others.length > 0 && (
                <div className="account-others">
                  {others.map((b) => (
                    <Money key={b.currency} minor={b.balance_minor} currency={b.currency} />
                  ))}
                </div>
              )}
            </div>
            <div className="row account-meta">
              <span>{t.accountKinds[account.kind]}</span>
              <span>{account.multi_currency ? t.accounts.many : account.currency}</span>
            </div>
          </div>

          <div className={`flip-face flip-back account account-${account.kind}`}>
            <div className="row">
              <span className="account-name">{t.accountBack.spentTitle[account.kind]}</span>
              <span className="badge">{account.multi_currency ? t.accounts.many : account.currency}</span>
            </div>
            <div className="account-spent">
              {spending.length === 0 && main !== undefined ? (
                <div className="account-balance">
                  <Money minor={0} currency={main.currency} />
                </div>
              ) : (
                spending.map((b) => (
                  <div key={b.currency} className="spent-row">
                    <div className="account-balance">
                      <Money minor={b.spent_minor} currency={b.currency} />
                    </div>
                    {shares && <span className="spent-share">{percent(b.base_spent_minor ?? 0, baseTotal)}%</span>}
                  </div>
                ))
              )}
            </div>
            {spending.length === 0 ? (
              <div className="account-note">{t.accountBack.noSpend}</div>
            ) : (
              <div className="account-foot">
                {converted && baseTotal !== null && (
                  <div className="account-note">
                    <Money minor={baseTotal} currency={baseCurrency} /> {t.accountBack.byRate}
                  </div>
                )}
                <button
                  type="button"
                  className="card-action"
                  onClick={(e) => {
                    e.stopPropagation() // кнопка не переворачивает карту
                    onReport()
                  }}
                >
                  {t.accountReport.open}
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <path d="M9 6l6 6-6 6" />
                  </svg>
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </Backdrop>
  )
}
