export const t = {
  loading: 'Загрузка…',
  openInTelegram: 'Откройте приложение из Telegram',
  personalBudget: 'Личный бюджет',
  retry: 'Повторить',
  save: 'Сохранить',
  cancel: 'Отмена',
  close: 'Закрыть',
  back: 'Назад',
  today: 'Сегодня',

  tabs: {
    operations: 'Операции',
    categories: 'Категории',
    reports: 'Отчёты',
    accounts: 'Счета',
    add: 'Добавить запись',
  },
  privacy: {
    hide: 'Скрыть суммы',
    show: 'Показать суммы',
  },
  theme: {
    open: 'Оформление',
    title: 'Оформление',
    label: 'Тема',
    modes: {
      auto: 'Как в Telegram',
      light: 'Светлая',
      dark: 'Тёмная',
    },
    hint: 'Режим «Как в Telegram» сам переключается вместе с темой мессенджера.',
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
    currencyMode: 'Валюты счёта',
    one: 'Одна валюта',
    many: 'Несколько валют',
    manyHint: 'Для карт и кошельков с несколькими валютами сразу (Revolut, Wise, Binance). Остаток считается отдельно по каждой.',
    mainCurrency: 'Основная валюта',
  },
  accountKinds: {
    debit: 'Дебетовая карта',
    credit: 'Кредитная карта',
    cash: 'Наличные',
  },
  accountBack: {
    spentTitle: {
      debit: 'Потрачено по карте',
      credit: 'Потрачено по карте',
      cash: 'Потрачено наличными',
    },
    spentNote: 'Все расходы с этого счёта за всё время. Переводы и пополнения не входят.',
    flipHint: 'Нажмите на карту, чтобы перевернуть',
    holdings: 'Остатки по валютам',
    noSpend: 'Расходов с этого счёта пока нет',
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
    noAccountInCurrency: 'Нет счёта в этой валюте: добавьте счёт на вкладке «Счета»',
    amount: 'Сумма',
    currency: 'Валюта',
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
    currencyLocked: 'У перевода и начального остатка валюта не меняется: она задана счетом.',
  },

  categories: {
    title: 'Категории',
    new: 'Новая категория',
    edit: 'Изменить категории',
    editDone: 'Готово',
    editTitle: 'Изменить категорию',
    name: 'Название',
    group: 'Группа',
    description: 'Описание',
    hidden: 'Скрыта',
    empty: 'В группе пока пусто',
    addExpense: 'Добавить трату в категорию',
    spendHere: 'Траты по категории',
    noSpend: 'Трат в этой категории пока нет',
  },
  groups: {
    critical: 'Критичные',
    debt: 'Долги',
    daily: 'Ежедневные',
    rainy: 'На чёрный день',
    goal: 'Цели накопления',
  },

  reports: {
    title: 'Отчёты',
    periods: {
      week: 'Неделя',
      twoWeeks: '2 недели',
      threeWeeks: '3 недели',
      month: 'Месяц',
      custom: 'Период',
    },
    from: 'С',
    to: 'По',
    allGroups: 'Все группы',
    byGroups: 'Группы',
    byCategories: 'Категории',
    spent: 'Потрачено',
    empty: 'За этот период трат нет',
    noRates: 'Суммы в разных валютах не складываются: курсов пока нет.',
    alsoSpent: 'Ещё потрачено в других валютах',
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
