import { useRef, useState, type PointerEvent, type ReactNode } from 'react'
import { haptic } from './haptic'
import { t } from './i18n'

const OPEN = -124 // на столько сдвигается строка, чтобы открыть кнопки
const INTENT = 8 // сколько пикселей по горизонтали считается жестом, а не нажатием

// Открытая строка одна на весь список: открывая вторую, первая закрывается.
let closeOpen: (() => void) | null = null

type Props = {
  children: ReactNode
  onEdit: () => void
  onDelete: () => void
}

// SwipeRow: смахивание влево открывает «Изменить» и «Удалить». Это дополнение: запись по-прежнему открывается нажатием,
// а удалить можно и в её форме. Вертикальная прокрутка не перехватывается: жест начинается, только если палец ведут вбок.
export function SwipeRow({ children, onEdit, onDelete }: Props) {
  const [offset, setOffset] = useState(0)
  const [dragging, setDragging] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const start = useRef<{ x: number; y: number; from: number } | null>(null)
  const moved = useRef(false)

  const open = offset <= OPEN / 2

  function close() {
    setOffset(0)
    setConfirming(false)
    if (closeOpen === close) closeOpen = null
  }

  function onDown(e: PointerEvent) {
    start.current = { x: e.clientX, y: e.clientY, from: offset }
    moved.current = false
  }

  function onMove(e: PointerEvent) {
    const s = start.current
    if (s === null) return
    const dx = e.clientX - s.x
    const dy = e.clientY - s.y
    if (!dragging) {
      if (Math.abs(dx) < INTENT || Math.abs(dx) < Math.abs(dy)) return
      if (closeOpen !== null && closeOpen !== close) closeOpen()
      ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
      setDragging(true)
    }
    moved.current = true
    setOffset(Math.min(0, Math.max(OPEN, s.from + dx)))
  }

  function onUp() {
    const s = start.current
    start.current = null
    if (s === null || !dragging) return
    setDragging(false)
    const next = offset <= OPEN / 2 ? OPEN : 0
    setOffset(next)
    if (next === OPEN) {
      closeOpen = close
      haptic('light')
    } else {
      setConfirming(false)
    }
  }

  return (
    <div className="swipe">
      <div className="swipe-acts" aria-hidden={!open}>
        {confirming ? (
          <button
            type="button"
            className="swipe-confirm"
            tabIndex={open ? 0 : -1}
            onClick={() => {
              haptic('warning')
              onDelete()
            }}
          >
            {t.swipe.confirm}
          </button>
        ) : (
          <>
            <button
              type="button"
              className="swipe-btn swipe-edit"
              aria-label={t.swipe.edit}
              tabIndex={open ? 0 : -1}
              onClick={() => {
                close()
                onEdit()
              }}
            >
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M4 20h4L19 9l-4-4L4 16zM13 7l4 4" />
              </svg>
            </button>
            <button type="button" className="swipe-btn swipe-del" aria-label={t.swipe.delete} tabIndex={open ? 0 : -1} onClick={() => setConfirming(true)}>
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M5 7h14M10 7V5h4v2M7 7l1 12h8l1-12M10 11v5M14 11v5" />
              </svg>
            </button>
          </>
        )}
      </div>
      <div
        className={dragging ? 'swipe-body swipe-drag' : 'swipe-body'}
        style={{ transform: `translateX(${offset}px)` }}
        onPointerDown={onDown}
        onPointerMove={onMove}
        onPointerUp={onUp}
        onPointerCancel={onUp}
        onClickCapture={(e) => {
          // Нажатие по сдвинутой строке закрывает её, а после жеста случайное нажатие не открывает запись.
          if (moved.current || open) {
            e.stopPropagation()
            e.preventDefault()
            moved.current = false
            if (open) close()
          }
        }}
      >
        {children}
      </div>
    </div>
  )
}
