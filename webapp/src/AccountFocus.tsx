import { useState } from 'react'
import type { Account } from './api'
import { t } from './i18n'
import { Backdrop } from './Modal'
import { Money } from './MoneyText'

type Props = {
  account: Account
  onClose: () => void
}

// Карточка счёта на переднем плане: всё остальное размыто. Нажатие переворачивает её,
// на обороте видно, сколько потрачено по этому счёту за всё время.
export function AccountFocus({ account, onClose }: Props) {
  const [flipped, setFlipped] = useState(false)

  return (
    <Backdrop onClose={onClose}>
      <div className="flip-scene">
        <div
          className={flipped ? 'flip flipped' : 'flip'}
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
            <div className="account-balance">
              <Money minor={account.balance_minor} currency={account.currency} />
            </div>
            <div className="row account-meta">
              <span>{t.accountKinds[account.kind]}</span>
              <span>{account.currency}</span>
            </div>
          </div>

          <div className={`flip-face flip-back account account-${account.kind}`}>
            <div className="row">
              <span className="account-name">{t.accountBack.spentTitle[account.kind]}</span>
              <span className="badge">{account.currency}</span>
            </div>
            <div className="account-balance">
              <Money minor={account.spent_minor} currency={account.currency} />
            </div>
            <div className="account-note">{t.accountBack.spentNote}</div>
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
