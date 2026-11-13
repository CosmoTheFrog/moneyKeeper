import { createContext, useContext } from 'react'
import { t } from './i18n'
import { Modal } from './Modal'
import { ChipPicker } from './Pickers'
import type { Theme } from './theme'

type ThemeState = {
  theme: Theme
  setTheme: (theme: Theme) => void
}

export const ThemeContext = createContext<ThemeState>({ theme: 'bento', setTheme: () => {} })

const options = (['bento', 'pop'] as const).map((m) => ({ value: m, label: t.theme.names[m] }))

export function ThemeSettings({ onClose }: { onClose: () => void }) {
  const { theme, setTheme } = useContext(ThemeContext)
  return (
    <Modal title={t.theme.title} onClose={onClose}>
      <ChipPicker label={t.theme.label} value={theme} options={options} onChange={setTheme} />
    </Modal>
  )
}
