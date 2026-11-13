import type { KeyboardEvent } from 'react'

// Нажатие Enter или пробела на элементе с role="button" работает как щелчок. Плитки сделаны не <button>, потому что внутри
// них лежат суммы, а сумма сама кнопка (скрывает цифры), и кнопка в кнопке недопустима.
export const onKey = (fn: () => void) => (e: KeyboardEvent<HTMLElement>) => {
  if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    fn()
  }
}
