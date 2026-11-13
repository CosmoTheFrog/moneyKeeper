export type ThemeMode = 'auto' | 'light' | 'dark'
export type Theme = 'light' | 'dark'

const KEY = 'mk.theme'

// По умолчанию светлая тема: «Бенто» рисуется светлым. «Как в Telegram» включается явно.
export function parseMode(raw: string | null): ThemeMode {
  return raw === 'dark' || raw === 'auto' ? raw : 'light'
}

// resolveTheme: явный выбор главнее, «авто» следует теме Telegram, без неё теме системы.
export function resolveTheme(mode: ThemeMode, telegramScheme: string | undefined, prefersDark: boolean): Theme {
  if (mode !== 'auto') return mode
  if (telegramScheme === 'dark' || telegramScheme === 'light') return telegramScheme
  return prefersDark ? 'dark' : 'light'
}

export function readMode(): ThemeMode {
  try {
    return parseMode(localStorage.getItem(KEY))
  } catch {
    return 'light' // хранилище может быть недоступно
  }
}

export function saveMode(mode: ThemeMode): void {
  try {
    localStorage.setItem(KEY, mode)
  } catch {
    // не сохранилось, тема живёт до закрытия
  }
}

export function applyTheme(theme: Theme): void {
  document.documentElement.dataset.theme = theme
}
