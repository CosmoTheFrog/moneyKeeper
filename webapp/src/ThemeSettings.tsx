import { createContext, useContext } from 'react'
import { t } from './i18n'
import { Modal } from './Modal'
import { ChipPicker } from './Pickers'
import type { ThemeMode } from './theme'

type ThemeState = {
  mode: ThemeMode
  setMode: (mode: ThemeMode) => void
}

export const ThemeContext = createContext<ThemeState>({ mode: 'auto', setMode: () => {} })

const options = (['auto', 'light', 'dark'] as const).map((m) => ({ value: m, label: t.theme.modes[m] }))

export function ThemeSettings({ onClose }: { onClose: () => void }) {
  const { mode, setMode } = useContext(ThemeContext)
  return (
    <Modal title={t.theme.title} onClose={onClose}>
      <ChipPicker label={t.theme.label} value={mode} options={options} onChange={setMode} />
      <p className="hint tight">{t.theme.hint}</p>
    </Modal>
  )
}
