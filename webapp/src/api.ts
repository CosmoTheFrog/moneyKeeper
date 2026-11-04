import type { CurrencyInfo } from './money'
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
  // Знаки после запятой и признак монеты по каждой валюте.
  currency_info: CurrencyInfo[]
}

// wallet и piggy (кошелёк, копилка) лежат вне бюджета.
export type AccountKind = 'cash' | 'debit' | 'credit' | 'wallet' | 'piggy'

// Остаток и расходы счёта в одной валюте. spent_minor положительная: расходы за всё время, без переводов и доходов.
// base_* это то же в базовой валюте бюджета: остаток по сегодняшнему курсу, расходы по курсу дня каждой записи.
// null, если нужного курса нет.
export type Balance = {
  currency: string
  balance_minor: number
  spent_minor: number
  base_balance_minor: number | null
  base_spent_minor: number | null
}

// currency это основная валюта счёта, в balances она первая. У счёта с multi_currency записи возможны в любой валюте.
export type Account = {
  id: number
  name: string
  kind: AccountKind
  currency: string
  multi_currency: boolean
  is_default: boolean
  off_budget: boolean
  // Закрытый счёт: в списках его нет, записи по нему остаются в истории.
  archived: boolean
  // Копилка: цель и конверт, из которого она пополняется.
  target_minor: number | null
  target_date: string | null
  category_id: number | null
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
  // Остаток категории (и перерасход) переносится в следующий месяц.
  rollover: boolean
}

// exchange это две записи в разных валютах, fee комиссия обмена со счёта вне бюджета.
export type TransactionKind = 'expense' | 'income' | 'transfer' | 'opening' | 'exchange' | 'fee'

// amount_minor со знаком: расход и уходящая нога перевода отрицательные.
// base_minor это сумма в базовой валюте по курсу дня записи, null если курса нет.
export type Transaction = {
  id: number
  account_id: number
  category_id: number | null
  kind: TransactionKind
  amount_minor: number
  currency: string
  base_minor: number | null
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
  // Комиссия сети при переводе между кошельками вне бюджета: приходит amount_minor минус комиссия.
  fee_minor?: number
}

export type NewExchange = {
  from_account_id: number
  to_account_id: number
  out_minor: number
  out_currency: string
  in_minor: number
  in_currency: string
  category_id?: number
  fee_minor?: number
  fee_category_id?: number
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
  // Файл уходит как есть: границу multipart браузер ставит сам.
  const isForm = body instanceof FormData
  if (body !== undefined && !isForm) headers['Content-Type'] = 'application/json'

  const resp = await fetch(path, { method, headers, body: body === undefined || isForm ? (body as FormData | undefined) : JSON.stringify(body) })
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

// inCurrency: валюта, в которую сервер пересчитывает суммы (поля base_*); пусто значит базовая валюта бюджета.
export const fetchAccounts = async (inCurrency: string) =>
  (await request<{ accounts: Account[] }>('GET', `/api/accounts?in=${inCurrency}`)).accounts

// Rate: сколько единиц валюты In стоит одна целая единица currency, десятичной строкой.
export type RateTable = { day: string; in: string; rates: { currency: string; rate: string }[] }

// Добавить монету (цена на Binance) или акцию (Yahoo Finance) по коду. Сервер проверяет цену и подбирает точность.
export const addAsset = (code: string, stock: boolean) => request<CurrencyInfo>('POST', '/api/assets', { code, stock })

export const fetchRates = (inCurrency: string) => request<RateTable>('GET', `/api/rates?in=${inCurrency}`)

export const createAccount = (name: string, kind: AccountKind, currency: string, multiCurrency: boolean, openingBalanceMinor: number) =>
  request<Account>('POST', '/api/accounts', { name, kind, currency, multi_currency: multiCurrency, opening_balance_minor: openingBalanceMinor })

// Копилка заводится вместе с конвертом в группе «Цели накопления»; цель необязательна.
export const createPiggy = (name: string, currency: string, targetMinor: number | null, targetDate: string | null) =>
  request<Account>('POST', '/api/accounts', { name, kind: 'piggy', currency, target_minor: targetMinor ?? undefined, target_date: targetDate ?? undefined })

export const fetchCategories = async () => (await request<{ categories: Category[] }>('GET', '/api/categories')).categories

export const fetchCategoryGroups = async () => (await request<{ groups: CategoryGroup[] }>('GET', '/api/category-groups')).groups

export const createCategoryGroup = (name: string) => request<CategoryGroup>('POST', '/api/category-groups', { name })

export const createCategory = (name: string, groupId: number) => request<Category>('POST', '/api/categories', { name, group_id: groupId })

// Бюджет месяца в базовой валюте. to_budget_minor накопительное: нераспределённое переходит в следующий месяц само.
// missing_rates число сумм без курса, в итоги они не вошли.
export type BudgetRow = { category_id: number; budgeted_minor: number; spent_minor: number; balance_minor: number }

export type BudgetMonth = {
  month: string
  currency: string
  income_minor: number
  budgeted_minor: number
  to_budget_minor: number
  missing_rates: number
  categories: BudgetRow[]
}

export const fetchBudgetMonth = (month: string) => request<BudgetMonth>('GET', `/api/budget/months/${month}`)

export const setBudgeted = (month: string, categoryId: number, budgetedMinor: number) =>
  request<BudgetMonth>('PUT', `/api/budget/months/${month}/categories/${categoryId}`, { budgeted_minor: budgetedMinor })

export const updateCategory = (id: number, patch: { name?: string; description?: string; hidden?: boolean; rollover?: boolean }) =>
  request<Category>('PATCH', `/api/categories/${id}`, patch)

// Сервер отдаёт не больше 200 записей за раз; отчёты считаются по всем, поэтому страницы дочитываются до неполной.
const PAGE = 200

export async function fetchTransactions(inCurrency: string): Promise<Transaction[]> {
  const all: Transaction[] = []
  for (;;) {
    const { transactions } = await request<{ transactions: Transaction[] }>('GET', `/api/transactions?limit=${PAGE}&offset=${all.length}&in=${inCurrency}`)
    all.push(...transactions)
    if (transactions.length < PAGE) return all
  }
}

export const createTransaction = (tx: NewTransaction) => request<{ transactions: Transaction[] }>('POST', '/api/transactions', tx)

export const updateTransaction = (id: number, patch: TransactionPatch) =>
  request<{ transactions: Transaction[] }>('PATCH', `/api/transactions/${id}`, patch)

// Закрыть можно пустой неосновной счёт; записи по нему остаются в истории.
export const archiveAccount = async (id: number): Promise<void> => {
  await send('POST', `/api/accounts/${id}/archive`)
}

export const deleteTransaction = async (id: number): Promise<void> => {
  await send('DELETE', `/api/transactions/${id}`)
}

export const fetchEvents = async (id: number) => (await request<{ events: TxEvent[] }>('GET', `/api/transactions/${id}/events`)).events

// currency необязательна: валюта формы, по ней считается точность числа, если в тексте валюты нет.
export const createExchange = (e: NewExchange) => request<{ transactions: Transaction[] }>('POST', '/api/exchanges', e)

// Позиция по монете: стоимость, цена покупки и прибыль в валюте портфеля. profit_bp в сотых долях процента.
export type PortfolioItem = {
  currency: string
  quantity_minor: number
  value_minor: number
  cost_minor: number
  profit_minor: number
  profit_bp: number | null
  realized_minor: number
  avg_price: string
}

export type Portfolio = { currency: string; missing_rates: number; items: PortfolioItem[] }

export const fetchPortfolio = (inCurrency: string) => request<Portfolio>('GET', `/api/portfolio?in=${inCurrency}`)

// Расписание платежей. amount_kind: exact, approx (допуск 7,5%) или range (до amount_max_minor). days: числа месяца,
// 0 это последний день. weekend_shift: none, back (на предыдущий будний день), forward (на следующий).
export type AmountKind = 'exact' | 'approx' | 'range'
export type WeekendShift = 'none' | 'back' | 'forward'

export type Schedule = {
  id: number
  name: string
  category_id: number
  account_id: number | null
  currency: string
  amount_kind: AmountKind
  amount_minor: number
  amount_max_minor: number | null
  every_months: number
  days: number[]
  start_month: string
  weekend_shift: WeekendShift
  remind_before: number
}

export type ScheduleInput = Omit<Schedule, 'id' | 'start_month'>

// status: upcoming впереди, overdue дата прошла без траты, paid нашлась подходящая трата, skipped пропущен.
export type Occurrence = {
  schedule: Schedule
  due_on: string
  status: 'upcoming' | 'overdue' | 'paid' | 'skipped'
  transaction_id: number | null
}

export const fetchOccurrences = async () => (await request<{ occurrences: Occurrence[] }>('GET', '/api/schedules/occurrences')).occurrences

export const createSchedule = (s: ScheduleInput) => request<Schedule>('POST', '/api/schedules', s)

export const updateSchedule = (id: number, s: ScheduleInput) => request<Schedule>('PUT', `/api/schedules/${id}`, s)

export const deleteSchedule = async (id: number): Promise<void> => {
  await send('DELETE', `/api/schedules/${id}`)
}

export const skipOccurrence = async (id: number, dueOn: string): Promise<void> => {
  await send('POST', `/api/schedules/${id}/skip`, { due_on: dueOn })
}

export const parseQuick = (text: string, currency: string) => request<ParsedInput>('POST', '/api/transactions/parse', { text, currency })

// Строка импорта: amount_minor со знаком, расход отрицательный. source_category заполнено, если категорию
// из файла не удалось найти по названию и её выбирает пользователь.
export type ImportRow = {
  date: string
  amount_minor: number
  currency: string
  description: string
  category_id: number | null
  source_category: string
  suggested: boolean
  duplicate: boolean
  // Задан у перевода, прочитанного из сети: по нему внесённый перевод не пишется второй раз.
  external_id: string
}

export type ImportPreview = { rows: ImportRow[]; skipped: number }

// Отчёт прежнего бота в xlsx: разбирается на сервере, ничего не сохраняет до commitImport.
export const previewImportFile = (accountId: number, file: File) => {
  const form = new FormData()
  form.append('account_id', String(accountId))
  form.append('file', file)
  return request<ImportPreview>('POST', '/api/import/file', form)
}

export const commitImport = (accountId: number, rows: ImportRow[]) =>
  request<{ saved: number }>('POST', '/api/import/commit', { account_id: accountId, rows })

// Публичный адрес кошелька, привязанный к счёту. consent это явное согласие отправить адрес в источник данных о сети.
export type Wallet = { id: number; chain: string; address: string }

export const fetchWallets = async (accountId: number) => (await request<{ wallets: Wallet[] }>('GET', `/api/accounts/${accountId}/wallets`)).wallets

export const addWallet = (accountId: number, chain: string, address: string, consent: boolean) =>
  request<Wallet>('POST', `/api/accounts/${accountId}/wallets`, { chain, address, consent })

export const removeWallet = async (id: number): Promise<void> => {
  await send('DELETE', `/api/wallets/${id}`)
}

// Переводы адреса за последние 90 дней, ничего не сохраняет до commitImport.
export const previewWallet = (id: number) => request<ImportPreview>('POST', `/api/wallets/${id}/preview`)
