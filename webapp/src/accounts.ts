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
