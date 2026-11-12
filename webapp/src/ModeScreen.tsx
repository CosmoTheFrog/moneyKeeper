import { useState } from 'react'
import { ApiError, setInterfaceMode, type InterfaceMode } from './api'
import { errorText, t } from './i18n'

type PickerProps = {
  value: InterfaceMode | null
  onPick: (mode: InterfaceMode) => void
}

// ModePicker: две карточки с описанием режимов. Общий для выбора при входе и для настроек.
export function ModePicker({ value, onPick }: PickerProps) {
  return (
    <div className="list" role="radiogroup" aria-label={t.mode.title}>
      {(['simple', 'full'] as const).map((m) => (
        <button key={m} type="button" role="radio" aria-checked={m === value} className={m === value ? 'option option-picked' : 'option'} onClick={() => onPick(m)}>
          <span className="option-name">
            <b>{t.mode[m].title}</b>
            <span className="hint tight">{t.mode[m].text}</span>
          </span>
        </button>
      ))}
    </div>
  )
}

type Props = {
  onDone: () => void
}

// ModeScreen: выбор режима интерфейса при первом входе, после выбора валюты. Сменить режим можно в «Ещё».
export function ModeScreen({ onDone }: Props) {
  const [picked, setPicked] = useState<InterfaceMode>('simple')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit() {
    setBusy(true)
    setError(null)
    try {
      await setInterfaceMode(picked)
      onDone()
    } catch (e) {
      setError(errorText(e instanceof ApiError ? e.code : 'internal'))
      setBusy(false)
    }
  }

  return (
    <main className="screen">
      <div className="step">{t.mode.step}</div>
      <h1>{t.mode.title}</h1>
      <p className="hint">{t.mode.hint}</p>
      <ModePicker value={picked} onPick={setPicked} />
      {error !== null && <p className="error">{error}</p>}
      <button type="button" className="primary" disabled={busy} onClick={() => void submit()}>
        {t.currency.next}
      </button>
    </main>
  )
}
