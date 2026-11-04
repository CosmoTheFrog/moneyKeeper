// Лёгкая отдача Telegram на сохранение, удаление и жесты. Вне Telegram, на старых клиентах и при ошибке молчит.
export function haptic(kind: 'light' | 'success' | 'warning'): void {
  try {
    const h = window.Telegram?.WebApp.HapticFeedback
    if (h === undefined) return
    if (kind === 'light') h.impactOccurred('light')
    else h.notificationOccurred(kind)
  } catch {
    // отдача не обязательна
  }
}
