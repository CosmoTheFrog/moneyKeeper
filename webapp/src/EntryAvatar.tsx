import type { CSSProperties } from 'react'
import { categoryIcon } from './categoryIcons'
import type { Entry } from './ledger'

const PATHS = {
  income: 'M3 8a3 3 0 013-3h11a2 2 0 012 2v2M3 8v9a3 3 0 003 3h13a1 1 0 001-1v-8a1 1 0 00-1-1H6a3 3 0 01-3-3zM16.5 14.5h.01',
  swap: 'M7 7h12M16 3l4 4-4 4M17 17H5M8 13l-4 4 4 4',
  fee: 'M12 3c-4 0-7 1.3-7 3v12c0 1.7 3 3 7 3s7-1.3 7-3V6c0-1.7-3-3-7-3zM5 12c0 1.7 3 3 7 3s7-1.3 7-3M5 6c0 1.7 3 3 7 3s7-1.3 7-3',
  opening: 'M5 21V4M5 4h11l-2 4 2 4H5',
}

type Props = {
  entry: Entry
  categoryName: string | undefined
  // Цвет категории (оттенок её группы); у доходов и переводов свой.
  color: string | undefined
}

// EntryAvatar: плитка слева в строке записи. Расход несёт значок и цвет своей категории, остальное свои значки.
export function EntryAvatar({ entry, categoryName, color }: Props) {
  let tint = 'var(--muted)'
  let path: string | null = null
  let letter = ''
  switch (entry.kind) {
    case 'expense':
      tint = color ?? 'var(--accent)'
      path = categoryName !== undefined ? categoryIcon(categoryName) : null
      letter = categoryName !== undefined ? ([...categoryName.trim()][0]?.toUpperCase() ?? '') : ''
      break
    case 'income':
      tint = 'var(--pos)'
      path = PATHS.income
      break
    case 'transfer':
    case 'exchange':
      path = PATHS.swap
      break
    case 'fee':
      path = PATHS.fee
      break
    case 'opening':
      path = PATHS.opening
      break
  }
  return (
    <span className="avatar" style={{ '--gc': tint } as CSSProperties} aria-hidden="true">
      {path !== null ? (
        <svg viewBox="0 0 24 24">
          <path d={path} />
        </svg>
      ) : (
        <b>{letter}</b>
      )}
    </span>
  )
}
