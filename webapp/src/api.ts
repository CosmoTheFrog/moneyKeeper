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

// Остаток и расходы счёта в одной валюте. spent_minor положительная: расходы за всё время, без переводов и доходов.
export type Balance = {
  currency: string
  balance_minor: number
  spent_minor: number
}

// currency это основная валюта счёта, в balances она первая. У счёта с multi_currency записи возможны в любой валюте.
export type Account = {
  id: number
  name: string
  kind: AccountKind
  currency: string
  multi_currency: boolean
  is_default: boolean
  balances: Balance[]
}

// Группа категорий приходит в порядке показа, название уже на языке пользователя.
export type CategoryGroup = {
  id: number
  name: string
}

export type Category = {
  id: number
  group_id: number
  name: string
  description: string
  hidden: boolean
}

export type TransactionKind = 'expense' | 'income' | 'transfer' | 'opening'

// amount_minor со знаком: расход и уходящая нога перевода отрицательные.
export type Transaction = {
  id: number
  account_id: number
  category_id: number | null
  kind: TransactionKind
  amount_minor: number
  currency: string
  date: string
  note: string
  transfer_id: number | null
}

export type Snapshot = {
  account_id: number
  category_id: number | null
  amount_minor: number
  date: string
  note: string
}

export type TxEvent = {
  id: number
  action: 'create' | 'update' | 'delete'
  at: string
  before: Snapshot | null
  after: Snapshot | null
}

export type NewTransaction = {
  kind: 'expense' | 'income' | 'transfer'
  account_id: number
  to_account_id?: number
  category_id?: number
  amount_minor: number
  currency?: string
  date?: string
  note: string
}

export type TransactionPatch = {
  account_id?: number
  category_id?: number
  amount_minor?: number
  currency?: string
  date?: string
  note?: string
}

export type ParsedInput = {
  amount_minor: number
  currency: string
  note: string
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

async function send(method: string, path: string, body?: unknown): Promise<Response> {
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
  return resp
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  return (await send(method, path, body)).json()
}

export const fetchMe = () => request<Me>('GET', '/api/me')

export const setBaseCurrency = (currency: string) => request<Budget>('PUT', '/api/budget/currency', { currency })

export const fetchAccounts = async () => (await request<{ accounts: Account[] }>('GET', '/api/accounts')).accounts

export const createAccount = (name: string, kind: AccountKind, currency: string, multiCurrency: boolean, openingBalanceMinor: number) =>
  request<Account>('POST', '/api/accounts', { name, kind, currency, multi_currency: multiCurrency, opening_balance_minor: openingBalanceMinor })

export const fetchCategories = async () => (await request<{ categories: Category[] }>('GET', '/api/categories')).categories

export const fetchCategoryGroups = async () => (await request<{ groups: CategoryGroup[] }>('GET', '/api/category-groups')).groups

export const createCategoryGroup = (name: string) => request<CategoryGroup>('POST', '/api/category-groups', { name })

export const createCategory = (name: string, groupId: number) => request<Category>('POST', '/api/categories', { name, group_id: groupId })

export const updateCategory = (id: number, patch: { name?: string; description?: string; hidden?: boolean }) =>
  request<Category>('PATCH', `/api/categories/${id}`, patch)

export const fetchTransactions = async (limit = 200) =>
  (await request<{ transactions: Transaction[] }>('GET', `/api/transactions?limit=${limit}`)).transactions

export const createTransaction = (tx: NewTransaction) => request<{ transactions: Transaction[] }>('POST', '/api/transactions', tx)

export const updateTransaction = (id: number, patch: TransactionPatch) =>
  request<{ transactions: Transaction[] }>('PATCH', `/api/transactions/${id}`, patch)

export const deleteTransaction = async (id: number): Promise<void> => {
  await send('DELETE', `/api/transactions/${id}`)
}

export const fetchEvents = async (id: number) => (await request<{ events: TxEvent[] }>('GET', `/api/transactions/${id}/events`)).events

export const parseQuick = (text: string) => request<ParsedInput>('POST', '/api/transactions/parse', { text })
