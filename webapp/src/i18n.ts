export const t = {
  loading: 'Загрузка…',
  openInTelegram: 'Откройте приложение из Telegram',
  personalBudget: 'Личный бюджет',
  retry: 'Повторить',

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
    save: 'Сохранить',
    cancel: 'Отмена',
  },
  accountKinds: {
    debit: 'Дебетовая карта',
    credit: 'Кредитная карта',
    cash: 'Наличные',
  },

  errors: {
    invalid_currency: 'Эта валюта недоступна',
    invalid_account: 'Проверьте название и тип счёта',
    base_currency_set: 'Базовая валюта уже выбрана',
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
