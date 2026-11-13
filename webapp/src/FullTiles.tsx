import { useContext, type CSSProperties } from 'react'
import type { Account, Occurrence } from './api'
import { CategoryColorContext } from './categoryColor'
import { t } from './i18n'
import { Money } from './MoneyText'
import { onKey } from './onKey'
import { diffDays } from './report'

type Props = {
  accounts: Account[]
  // Ближайший неоплаченный платёж (просроченные первыми), null если платежей нет.
  nextPayment: Occurrence | null
  today: string
  onOpenAccounts: () => void
  onOpenSchedules: () => void
  onOpenSavings: () => void
}

// «Через 2 дня», «сегодня», «просрочен на 3 дн.».
function dueText(due: string, today: string): string {
  const n = diffDays(today, due)
  if (n === 0) return t.fullTiles.today
  if (n > 0) return `${t.fullTiles.in} ${n} ${t.fullTiles.days}`
  return `${t.fullTiles.overdue} ${-n} ${t.fullTiles.days}`
}

// FullTiles: плитки режима «С картами» на главном экране: лента счетов, ближайший платёж и копилка. Каждая плитка ведёт на свой экран.
export function FullTiles({ accounts, nextPayment, today, onOpenAccounts, onOpenSchedules, onOpenSavings }: Props) {
  useContext(CategoryColorContext)
  const cards = accounts.filter((a) => a.kind !== 'piggy')
  const piggy = accounts.find((a) => a.kind === 'piggy')
  const saved = piggy?.balances[0]?.balance_minor ?? 0
  const share = piggy?.target_minor ? Math.min(saved / piggy.target_minor, 1) : 0
  const RING = 2 * Math.PI * 14

  return (
    <>
      {cards.length > 0 && (
        <div className="mini-cards" aria-label={t.accounts.title}>
          {cards.map((a) => (
            <div key={a.id} role="button" tabIndex={0} className={`mini-card mini-${a.kind}`} onClick={onOpenAccounts} onKeyDown={onKey(onOpenAccounts)}>
              <span className="mini-name">{a.name}</span>
              <Money minor={a.balances[0]?.balance_minor ?? 0} currency={a.currency} />
            </div>
          ))}
        </div>
      )}
      <div className="bento bento-full">
        <div role="button" tabIndex={0} className="bt bt-pay bt-tap" onClick={onOpenSchedules} onKeyDown={onKey(onOpenSchedules)}>
          <span className="bt-cap">{t.fullTiles.nextPayment}</span>
          {nextPayment === null ? (
            <b className="bt-empty">{t.fullTiles.noPayments}</b>
          ) : (
            <>
              <b className="bt-name">{nextPayment.schedule.name}</b>
              <Money minor={nextPayment.schedule.amount_minor} currency={nextPayment.schedule.currency} />
              <span className={nextPayment.status === 'overdue' ? 'bt-due bt-due-over' : 'bt-due'}>{dueText(nextPayment.due_on, today)}</span>
            </>
          )}
        </div>
        <div role="button" tabIndex={0} className="bt bt-piggy bt-tap" onClick={onOpenSavings} onKeyDown={onKey(onOpenSavings)}>
          <span className="bt-top">
            <span className="bt-cap">{t.fullTiles.piggy}</span>
            {piggy !== undefined && piggy.target_minor !== null && (
              <svg width="36" height="36" viewBox="0 0 38 38" role="img" aria-label={`${Math.round(share * 100)}%`} style={{ '--share': share } as CSSProperties}>
                <circle cx="19" cy="19" r="14" fill="none" stroke="#fff" strokeWidth="6" />
                <circle cx="19" cy="19" r="14" fill="none" stroke="#8a6cff" strokeWidth="6" strokeLinecap="round" strokeDasharray={`${share * RING} ${RING}`} transform="rotate(-90 19 19)" />
              </svg>
            )}
          </span>
          {piggy === undefined ? (
            <b className="bt-empty">{t.fullTiles.noPiggy}</b>
          ) : (
            <>
              <span className="bt-sub">{piggy.name}</span>
              <Money minor={saved} currency={piggy.currency} />
            </>
          )}
        </div>
      </div>
    </>
  )
}
