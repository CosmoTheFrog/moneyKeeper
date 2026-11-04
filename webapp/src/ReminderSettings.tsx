import { useState } from 'react'
import { ApiError, setReminderHour } from './api'
import { errorText, t } from './i18n'
import { Modal } from './Modal'
import { ChipPicker } from './Pickers'

type Props = {
  // Выбранный час или null, если напоминание выключено.
  hour: number | null
  onChanged: () => void
  onClose: () => void
}

const DEFAULT_HOUR = 21
const HOURS = Array.from({ length: 24 }, (_, h) => h)
const modes = [
  { value: 'off', label: t.reminder.off },
  { value: 'on', label: t.reminder.on },
] as const

// ReminderSettings: ежедневное напоминание внести траты в Telegram. По умолчанию выключено.
export function ReminderSettings({ hour, onChanged, onClose }: Props) {
  const [value, setValue] = useState(hour)
  const [error, setError] = useState<string | null>(null)

  async function save(next: number | null) {
    setError(null)
    try {
      await setReminderHour(next)
      setValue(next)
      onChanged()
    } catch (e) {
      setError(errorText(e instanceof ApiError ? e.code : 'internal'))
    }
  }

  return (
    <Modal title={t.reminder.title} onClose={onClose}>
      <ChipPicker label={t.reminder.label} value={value === null ? 'off' : 'on'} options={[...modes]} onChange={(m) => void save(m === 'on' ? DEFAULT_HOUR : null)} />
      {value !== null && (
        <label className="field">
          <span className="label">{t.reminder.hour}</span>
          <select value={value} onChange={(e) => void save(Number(e.target.value))}>
            {HOURS.map((h) => (
              <option key={h} value={h}>
                {String(h).padStart(2, '0')}:00
              </option>
            ))}
          </select>
        </label>
      )}
      {error !== null && <p className="error">{error}</p>}
      <p className="hint tight">{t.reminder.hint}</p>
    </Modal>
  )
}
