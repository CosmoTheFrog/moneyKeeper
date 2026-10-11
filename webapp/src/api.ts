// TODO: типы генерируются из Go (internal/web/apidto), пока описаны вручную.
export type Budget = {
  id: number
  name: string
  base_currency: string | null
}

export type Me = {
  id: number
  first_name: string
  timezone: string
  budget: Budget
  base_currencies: string[]
  currencies: string[]
}

export type AccountKind = 'cash' | 'debit' | 'credit'

export type Account = {
  id: number
  name: string
  kind: AccountKind
  currency: string
  is_default: boolean
}

export class ApiError extends Error {
  readonly code: string
  readonly status: number

  constructor(code: string, status: number) {
    super(`${status} ${code}`)
    this.code = code
    this.status = status
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {
    Authorization: `tma ${window.Telegram?.WebApp.initData ?? ''}`,
    'X-Timezone': Intl.DateTimeFormat().resolvedOptions().timeZone,
  }
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  const resp = await fetch(path, { method, headers, body: body === undefined ? undefined : JSON.stringify(body) })
  if (!resp.ok) {
    const err: { code?: string } = await resp.json().catch(() => ({}))
    throw new ApiError(err.code ?? 'internal', resp.status)
  }
  return resp.json()
}

export const fetchMe = () => request<Me>('GET', '/api/me')

export const setBaseCurrency = (currency: string) => request<Budget>('PUT', '/api/budget/currency', { currency })

export const fetchAccounts = async () => (await request<{ accounts: Account[] }>('GET', '/api/accounts')).accounts

export const createAccount = (name: string, kind: AccountKind, currency: string) =>
  request<Account>('POST', '/api/accounts', { name, kind, currency })
