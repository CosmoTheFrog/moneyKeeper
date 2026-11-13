// Оформление: «Бенто» (светлые плитки, по умолчанию) и «Поп» (жёлтый фон, чёрные контуры, жёсткие тени).
export type Theme = 'bento' | 'pop'

const KEY = 'mk.theme'

// parseTheme: всё незнакомое, в том числе прежние «светлая», «тёмная» и «как в Telegram», это «Бенто».
export function parseTheme(raw: string | null): Theme {
  return raw === 'pop' ? 'pop' : 'bento'
}

export function readTheme(): Theme {
  try {
    return parseTheme(localStorage.getItem(KEY))
  } catch {
    return 'bento' // хранилище может быть недоступно
  }
}

export function saveTheme(theme: Theme): void {
  try {
    localStorage.setItem(KEY, theme)
  } catch {
    // не сохранилось, оформление живёт до закрытия
  }
}

export function applyTheme(theme: Theme): void {
  document.documentElement.dataset.theme = theme
}
