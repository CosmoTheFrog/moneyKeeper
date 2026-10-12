import { useEffect, useRef, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { t } from './i18n'

// Открытые слои друг над другом: Esc закрывает только верхний.
const open: symbol[] = []

type BackdropProps = {
  onClose: () => void
  children: ReactNode
  // sheet прижимает содержимое к низу экрана, по умолчанию оно по центру.
  sheet?: boolean
}

// Backdrop: размытый фон на весь экран, всё остальное становится фоном. Закрывается по Esc и по нажатию на фон.
// Рисуется в body, поэтому вложенные слои не обрезаются родителем.
export function Backdrop({ onClose, children, sheet = false }: BackdropProps) {
  const id = useRef(Symbol('backdrop'))
  const close = useRef(onClose)
  close.current = onClose

  useEffect(() => {
    const me = id.current
    open.push(me)
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && open[open.length - 1] === me) close.current()
    }
    window.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('keydown', onKey)
      open.splice(open.indexOf(me), 1)
    }
  }, [])

  return createPortal(
    <div
      className={sheet ? 'backdrop backdrop-sheet' : 'backdrop'}
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      {children}
    </div>,
    document.body,
  )
}

type ModalProps = {
  title: string
  onClose: () => void
  children: ReactNode
  sheet?: boolean
}

// Modal: окно с заголовком и крестиком справа сверху поверх размытого фона.
export function Modal({ title, onClose, children, sheet = false }: ModalProps) {
  return (
    <Backdrop onClose={onClose} sheet={sheet}>
      <div className="modal" role="dialog" aria-modal="true" aria-label={title}>
        <header className="modal-head">
          <h2 className="modal-title">{title}</h2>
          <button type="button" className="modal-x" aria-label={t.close} onClick={onClose}>
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" aria-hidden="true">
              <path d="M6 6l12 12M18 6L6 18" />
            </svg>
          </button>
        </header>
        <div className="modal-body">{children}</div>
      </div>
    </Backdrop>
  )
}
