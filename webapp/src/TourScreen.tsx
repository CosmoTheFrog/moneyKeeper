import { useState } from 'react'
import { ApiError, setTourSeen } from './api'
import { errorText, t } from './i18n'

type Props = {
  // Простой режим: в обучении нет счетов, копилок и портфеля.
  simple?: boolean
  onDone: () => void
}

// TourScreen: обучение при первом входе, по карточке на каждую часть приложения. Показывается, пока флаг не сохранён.
export function TourScreen({ simple = false, onDone }: Props) {
  const [index, setIndex] = useState(0)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const cards = simple ? t.tour.simpleCards : t.tour.cards
  const card = cards[index]
  const last = index === cards.length - 1

  async function finish() {
    setBusy(true)
    setError(null)
    try {
      await setTourSeen(true)
      onDone()
    } catch (e) {
      setError(errorText(e instanceof ApiError ? e.code : 'internal'))
      setBusy(false)
    }
  }

  if (card === undefined) return null
  return (
    <main className="screen">
      <header className="row">
        <span className="step">
          {t.tour.step} {index + 1}/{cards.length}
        </span>
        {!last && (
          <button type="button" className="chip" disabled={busy} onClick={() => void finish()}>
            {t.tour.skip}
          </button>
        )}
      </header>
      <div className={`tour-tile tour-tile-${index % 4}`} aria-hidden="true">
        {index + 1}
      </div>
      <h1>{card.title}</h1>
      <p className="hint">{card.text}</p>
      <ul className="tour-points">
        {card.points.map((p) => (
          <li key={p}>{p}</li>
        ))}
      </ul>
      {error !== null && <p className="error">{error}</p>}
      <button type="button" className="primary" disabled={busy} onClick={() => (last ? void finish() : setIndex(index + 1))}>
        {last ? t.tour.start : t.tour.next}
      </button>
    </main>
  )
}
