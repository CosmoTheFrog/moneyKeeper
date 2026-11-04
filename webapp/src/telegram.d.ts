type TelegramWebApp = {
  initData: string
  colorScheme?: string
  ready: () => void
  HapticFeedback?: {
    impactOccurred: (style: 'light' | 'medium' | 'heavy' | 'rigid' | 'soft') => void
    notificationOccurred: (type: 'error' | 'success' | 'warning') => void
  }
}

// Расширение глобального Window возможно только через interface.
declare global {
  interface Window {
    Telegram?: { WebApp: TelegramWebApp }
  }
}

export {}
