import { useState } from 'react'
import type { Account } from './api'
import { t } from './i18n'
import { Backdrop } from './Modal'
import { Money } from './MoneyText'

type Props = {
  account: Account
  onClose: () => void
}

// Карточка счёта на переднем плане: всё остальное размыто. Нажатие переворачивает её.
// На обороте расходы по этому счёту за всё время, для счёта с несколькими валютами отдельно по каждой.
export function AccountFocus({ account, onClose }: Props) {
  const [flipped, setFlipped] = useState(false)
  const [main, ...others] = account.balances
  const spending = account.balances.filter((b) => b.spent_minor > 0)
  // Карточка растёт вместе с числом валют: у обычного счёта это 210, у многовалютного больше.
  const rows = Math.max(account.balances.length, spending.length, 1)
  const height = Math.max(210, 150 + 34 * rows)

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
                  <div key={b.currency} className="account-balance">
                    <Money minor={b.spent_minor} currency={b.currency} />
                  </div>
                ))
              )}
            </div>
            <div className="account-note">{spending.length === 0 ? t.accountBack.noSpend : t.accountBack.spentNote}</div>
          </div>
        </div>
        <p className="flip-hint">{t.accountBack.flipHint}</p>
        <button type="button" className="flip-close" onClick={onClose}>
          {t.close}
        </button>
      </div>
    </Backdrop>
  )
}
