type TelegramWebApp = {
  initData: string
  colorScheme?: string
  ready: () => void
}

// Расширение глобального Window возможно только через interface.
declare global {
  interface Window {
    Telegram?: { WebApp: TelegramWebApp }
  }
}

export {}
