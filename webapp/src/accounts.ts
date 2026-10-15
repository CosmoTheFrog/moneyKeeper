import type { Account } from './api'

// Можно ли вести на счёте записи в валюте currency: на обычном только в его валюте, на многовалютном в любой.
export function allows(account: Account, currency: string): boolean {
  return account.multi_currency || account.currency === currency
}

export function balanceIn(account: Account, currency: string): number {
  return account.balances.find((b) => b.currency === currency)?.balance_minor ?? 0
}

// usableCurrencies: валюты, в которых есть куда записать. Многовалютный счёт открывает любую из all.
export function usableCurrencies(accounts: Account[], all: string[]): string[] {
  if (accounts.some((a) => a.multi_currency)) return all
  return all.filter((c) => accounts.some((a) => a.currency === c))
}

// pickAccount выбирает счёт для валюты: текущий, если он её допускает, иначе основной, иначе первый подходящий.
// Нет подходящего счёта: 0.
export function pickAccount(accounts: Account[], currency: string, currentId: number): number {
  const current = accounts.find((a) => a.id === currentId)
  if (current !== undefined && allows(current, currency)) return current.id
  const fitting = accounts.filter((a) => allows(a, currency))
  return (fitting.find((a) => a.is_default) ?? fitting[0])?.id ?? 0
}

export type Holdings = {
  // Итог в валюте показа; null, если хотя бы у одной валюты нет курса.
  total: number | null
  // Самые крупные остатки (по стоимости в валюте показа), без нулевых.
  top: Account['balances']
  // Сколько остатков не поместилось в top.
  more: number
}

// holdings сводит кошелёк с десятком монет в итог и несколько крупнейших остатков. Минусовой остаток (ушли в минус по монете)
// вычитается из итога как есть, а по размеру сравнивается по модулю.
export function holdings(account: Account, limit: number): Holdings {
  const nonZero = account.balances.filter((b) => b.balance_minor !== 0)
  const known = nonZero.every((b) => b.base_balance_minor !== null)
  const total = known ? nonZero.reduce((s, b) => s + (b.base_balance_minor ?? 0), 0) : null
  const size = (b: Account['balances'][number]) => Math.abs(b.base_balance_minor ?? 0)
  const sorted = [...nonZero].sort((a, b) => size(b) - size(a) || a.currency.localeCompare(b.currency))
  return { total, top: sorted.slice(0, limit), more: Math.max(sorted.length - limit, 0) }
}
