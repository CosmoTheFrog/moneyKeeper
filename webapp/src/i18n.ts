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
    byRate: 'по курсу дня каждой траты',
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
    newGroup: 'Новая группа',
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

  reports: {
    title: 'Отчёты',
    periods: {
      week: 'Неделя',
      twoWeeks: '2 нед.',
      threeWeeks: '3 нед.',
      month: 'Месяц',
      all: 'Всё время',
      custom: 'Свой период',
    },
    allGroups: 'Все группы',
    byGroups: 'Группы',
    byCategories: 'Категории',
    spent: 'Потрачено',
    empty: 'За этот период трат нет',
    inBase: 'Всё в',
    inBaseHint: 'Каждая трата пересчитана по курсу дня, когда она была записана.',
    ownCurrency: 'Только траты в этой валюте, без пересчёта.',
    missingRates: 'Без курса, в сумму не вошло',
    alsoSpent: 'Ещё потрачено в других валютах',
    perDay: 'В среднем в день',
    count: 'Трат',
  },

  calendar: {
    title: 'Период',
    apply: 'Показать',
    pickStart: 'Выберите первый день',
    pickEnd: 'Теперь последний день',
    prev: 'Предыдущий месяц',
    next: 'Следующий месяц',
    months: ['Январь', 'Февраль', 'Март', 'Апрель', 'Май', 'Июнь', 'Июль', 'Август', 'Сентябрь', 'Октябрь', 'Ноябрь', 'Декабрь'],
    monthsShort: ['янв', 'фев', 'мар', 'апр', 'мая', 'июн', 'июл', 'авг', 'сен', 'окт', 'ноя', 'дек'],
    weekdays: ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'],
    days: ['день', 'дня', 'дней'] as const,
  },

  accountReport: {
    open: 'Куда ушли деньги',
    title: 'Расходы по счёту',
    where: 'Куда',
    when: 'Когда',
    list: 'Все траты',
    noSpend: 'Расходов с этого счёта за период нет',
    noteTransfers: 'Переводы и пополнения не входят.',
  },

  errors: {
    invalid_currency: 'Эта валюта недоступна',
    invalid_account: 'Проверьте название, тип и сумму счёта',
    invalid_category: 'Проверьте название категории или группы, оно должно быть уникальным',
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

// plural выбирает форму слова по числу: 1 день, 2 дня, 5 дней, 11 дней, 21 день.
export function plural(n: number, forms: readonly [string, string, string]): string {
  const mod100 = n % 100
  const mod10 = n % 10
  if (mod100 >= 11 && mod100 <= 14) return forms[2]
  if (mod10 === 1) return forms[0]
  if (mod10 >= 2 && mod10 <= 4) return forms[1]
  return forms[2]
}

export function currencyName(code: string): string {
  return hasKey(t.currencyNames, code) ? t.currencyNames[code] : code
}
