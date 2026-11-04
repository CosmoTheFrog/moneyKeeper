import { useContext, type ReactNode } from 'react'
import type { Account, Category } from './api'
import { EntryAvatar } from './EntryAvatar'
import { CategoryColorContext } from './groupColor'
import { dayNet, groupByDay, type Entry } from './ledger'
import { t } from './i18n'
import { Money } from './MoneyText'
import { currencySign } from './money'

type Props = {
  entries: Entry[]
  accounts: Account[]
  categories: Category[]
  today: string
  onOpen: (entry: Entry) => void
  // Для трат заголовок это заметка («аренда за октябрь»), а название категории запасное.
  // withCategory дописывает категорию в подпись: нужно там, где в списке разные категории.
  noteFirst?: 'plain' | 'withCategory'
  // Валюта итогов дня; без неё итог в заголовке дня не показывается.
  totalCurrency?: string
  // Нажатие на заголовок дня открывает день.
  onOpenDay?: (date: string) => void
  // Оборачивает строку записи, например в смахивание.
  wrapRow?: (entry: Entry, row: ReactNode) => ReactNode
  // Без заголовков дней: список одного дня на его экране.
  bare?: boolean
}

function dayLabel(date: string, today: string): string {
  if (date === today) return t.today
  const [y, m, d] = date.split('-').map(Number)
  return new Date(y ?? 1970, (m ?? 1) - 1, d ?? 1).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' })
}

// Список записей по дням. Нажатие на строку открывает запись, нажатие на сумму скрывает суммы.
export function EntryList({ entries, accounts, categories, today, onOpen, noteFirst, totalCurrency, onOpenDay, wrapRow, bare = false }: Props) {
  const colors = useContext(CategoryColorContext)
  const accountName = new Map(accounts.map((a) => [a.id, a.name]))
  const categoryName = new Map(categories.map((c) => [c.id, c.name]))

  function title(e: Entry): string {
    switch (e.kind) {
      case 'transfer':
        return `${accountName.get(e.accountId) ?? ''} → ${e.toAccountId === null ? '…' : (accountName.get(e.toAccountId) ?? '')}`
      case 'exchange':
        return `${t.kinds.exchange} ${currencySign(e.currency)} → ${currencySign(e.inCurrency ?? e.currency)}`
      case 'fee':
        return t.kinds.fee
      case 'opening':
        return t.ledger.opening
      case 'income':
        return t.ledger.income
      case 'expense': {
        if (noteFirst !== undefined && e.note !== '') return e.note
        return (e.categoryId !== null ? categoryName.get(e.categoryId) : undefined) ?? t.ledger.uncategorized
      }
    }
  }

  function subtitle(e: Entry): string {
    if (noteFirst !== undefined && e.kind === 'expense') {
      const category = noteFirst === 'withCategory' && e.note !== '' && e.categoryId !== null ? (categoryName.get(e.categoryId) ?? '') : ''
      return [accountName.get(e.accountId) ?? '', category].filter((s) => s !== '').join(' · ')
    }
    if (e.kind === 'exchange') {
      const route = `${accountName.get(e.accountId) ?? ''} → ${e.toAccountId === null ? '…' : (accountName.get(e.toAccountId) ?? '')}`
      return [route, e.note].filter((s) => s !== '').join(' · ')
    }
    const account = e.kind === 'transfer' ? '' : (accountName.get(e.accountId) ?? '')
    return [account, e.note].filter((s) => s !== '').join(' · ')
  }

  return (
    <>
      {groupByDay(entries).map((g) => {
        const net = totalCurrency === undefined ? null : dayNet(g.entries)
        const head = (
          <>
            <span>{dayLabel(g.date, today)}</span>
            <span className="day-end">
              {net !== null && totalCurrency !== undefined && <Money minor={net} currency={totalCurrency} signed />}
              {onOpenDay !== undefined && (
                <span className="chev" aria-hidden="true">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.6" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M9 6l6 6-6 6" />
                  </svg>
                </span>
              )}
            </span>
          </>
        )
        return (
          <section key={g.date}>
            {bare ? null : onOpenDay !== undefined ? (
              <div className="day day-row day-tap" role="button" tabIndex={0} onClick={() => onOpenDay(g.date)} onKeyDown={(ev) => (ev.key === 'Enter' || ev.key === ' ') && onOpenDay(g.date)}>
                {head}
              </div>
            ) : (
              <h2 className="day day-row">{head}</h2>
            )}
            <div className="card list-card">
              {g.entries.map((e) => {
                const row = (
                  <div className="entry entry-av">
                    <EntryAvatar entry={e} categoryName={e.categoryId !== null ? categoryName.get(e.categoryId) : undefined} color={e.categoryId !== null ? colors.get(e.categoryId) : undefined} />
                    <button type="button" className="entry-open" onClick={() => onOpen(e)}>
                      <span className="entry-title">{title(e)}</span>
                      <span className="entry-sub">{subtitle(e)}</span>
                    </button>
                    {e.kind === 'exchange' ? (
                      <span className="entry-amounts">
                        <Money minor={-e.amountMinor} currency={e.currency} />
                        {e.inMinor !== null && e.inCurrency !== null && <Money minor={e.inMinor} currency={e.inCurrency} signed />}
                      </span>
                    ) : (
                      <Money minor={e.amountMinor} currency={e.currency} signed={e.kind !== 'transfer'} />
                    )}
                  </div>
                )
                return <div key={e.id} className="entry-wrap">{wrapRow !== undefined ? wrapRow(e, row) : row}</div>
              })}
            </div>
          </section>
        )
      })}
    </>
  )
}
