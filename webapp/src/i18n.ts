export const t = {
  loading: 'Загрузка…',
  openInTelegram: 'Откройте приложение из Telegram',
  personalBudget: 'Личный бюджет',
  retry: 'Повторить',
  save: 'Сохранить',
  cancel: 'Отмена',
  today: 'Сегодня',

  tabs: {
    operations: 'Операции',
    categories: 'Категории',
    accounts: 'Счета',
    add: 'Добавить запись',
  },
  privacy: {
    hide: 'Скрыть суммы',
    show: 'Показать суммы',
  },

  currency: {
    step: 'Шаг 1 из 2',
    title: 'В какой валюте вы ведёте бюджет?',
    hint: 'Остальные валюты можно добавлять потом, суммы покажем по текущему курсу.',
    next: 'Продолжить',
  },
  currencyNames: {
    GEL: 'Грузинский лари',
    EUR: 'Евро',
    USD: 'Доллар США',
    RUB: 'Российский рубль',
    USDT: 'Tether',
  } satisfies Record<string, string>,

  accounts: {
    title: 'Счета',
    add: 'Добавить счёт',
    empty: 'Добавьте первый счёт: карту или наличные.',
    defaultBadge: 'основной',
    name: 'Название',
    namePlaceholder: 'Например, TBC или Наличные',
    kind: 'Тип',
    currency: 'Валюта',
    opening: 'Сколько денег на счёте сейчас',
    openingHint: 'Эти деньги попадут в «К распределению» как доход.',
  },
  accountKinds: {
    debit: 'Дебетовая карта',
    credit: 'Кредитная карта',
    cash: 'Наличные',
  },

  ledger: {
    title: 'Операции',
    empty: 'Записей пока нет. Нажмите «+», чтобы добавить первую.',
    opening: 'Начальный остаток',
    income: 'Доход',
    transfer: 'Перевод',
    uncategorized: 'Без категории',
  },
  kinds: {
    expense: 'Расход',
    income: 'Доход',
    transfer: 'Перевод',
  },

  add: {
    title: 'Новая запись',
    quick: 'Быстрый ввод',
    quickPlaceholder: 'Например, кофе 5 EUR',
    understood: 'Понял так',
    noAccountInCurrency: 'Нет счёта в этой валюте: выберите счёт сами',
    amount: 'Сумма',
    account: 'Счёт',
    fromAccount: 'Откуда',
    toAccount: 'Куда',
    category: 'Категория',
    pickCategory: 'Выберите категорию',
    date: 'Дата',
    dateHint: 'Пусто означает сегодня',
    note: 'Заметка',
    transferHint: 'Запиши, за что скидываешь: это поможет при анализе (100% забудешь).',
    noTarget: 'Для перевода нужен второй счёт в той же валюте',
  },

  edit: {
    title: 'Запись',
    delete: 'Удалить',
    confirmDelete: 'Точно удалить?',
    history: 'История изменений',
    hideHistory: 'Скрыть историю',
    created: 'Создана',
    updated: 'Изменена',
    deleted: 'Удалена',
    amountChanged: 'Сумма',
    dateChanged: 'Дата',
    noteChanged: 'Заметка',
    accountChanged: 'Счёт',
    categoryChanged: 'Категория',
  },

  categories: {
    title: 'Категории',
    add: 'Новая категория',
    name: 'Название',
    group: 'Группа',
    description: 'Описание',
    hidden: 'Скрыта',
    empty: 'В группе пока пусто',
  },
  groups: {
    critical: 'Критичные',
    debt: 'Долги',
    daily: 'Ежедневные',
    rainy: 'На чёрный день',
    goal: 'Цели накопления',
  },

  errors: {
    invalid_currency: 'Эта валюта недоступна',
    invalid_account: 'Проверьте название, тип и сумму счёта',
    invalid_category: 'Проверьте название категории, оно должно быть уникальным',
    invalid_transaction: 'Проверьте сумму, счёт и категорию',
    invalid_date: 'Неверная дата',
    invalid_amount: 'Неверная сумма: до двух знаков после запятой',
    currency_mismatch: 'У счетов должна быть одна валюта',
    no_amount: 'Не нашёл сумму в тексте',
    base_currency_set: 'Базовая валюта уже выбрана',
    not_found: 'Запись не найдена, обновите экран',
    unauthorized: 'Не удалось войти. Откройте приложение из Telegram заново',
    internal: 'Что-то пошло не так. Попробуйте ещё раз',
  } satisfies Record<string, string>,
}

function hasKey<T extends object>(obj: T, key: string): key is Extract<keyof T, string> {
  return key in obj
}

export function errorText(code: string): string {
  return hasKey(t.errors, code) ? t.errors[code] : t.errors.internal
}

export function currencyName(code: string): string {
  return hasKey(t.currencyNames, code) ? t.currencyNames[code] : code
}
