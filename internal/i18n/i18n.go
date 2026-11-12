// Package i18n хранит тексты, которые бэкенд сам кладёт в данные пользователя
// (стартовые категории). Тексты интерфейса живут в каталоге фронта.
package i18n

import (
	"fmt"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/money"
)

// DefaultAccountName это название скрытого счёта, который заводится в простом режиме, русский язык.
const DefaultAccountName = "Основной"

// DefaultCategories — стартовый набор категорий из README, русский язык.
func DefaultCategories() []models.DefaultCategory {
	return []models.DefaultCategory{
		{Name: "Аренда и ипотека", Description: "Жильё: аренда или платёж по ипотеке"},
		{Name: "Коммунальные услуги", Description: "Свет, вода, газ, отопление"},
		{Name: "Связь и интернет", Description: "Домашний интернет и мобильная связь"},
		{Name: "Страховки", Description: "Медицинская, автомобильная, жилья"},
		{Name: "Налоги", Description: "Обязательные налоги и сборы"},
		{Name: "Погашение долгов", Description: "Платежи по кредитам, картам и займам"},
		{Name: "Продукты", Description: "Еда и товары для дома"},
		{Name: "Питание вне дома", Description: "Кафе, рестораны, кофе, доставка"},
		{Name: "Транспорт и бензин", Description: "Проезд, такси, топливо"},
		{Name: "Здоровье", Description: "Аптека, врачи, анализы"},
		{Name: "Дом и быт", Description: "Бытовая химия, мелкая мебель, хозтовары"},
		{Name: "Одежда и обувь", Description: "Одежда и обувь"},
		{Name: "Личный уход", Description: "Косметика, парикмахерская, гигиена"},
		{Name: "Развлечения", Description: "Кино, игры, хобби, отдых"},
		{Name: "Подписки", Description: "Сервисы по подписке: музыка, видео, приложения"},
		{Name: "Подарки", Description: "Подарки близким и на праздники"},
		{Name: "Образование", Description: "Курсы, книги, обучение"},
		{Name: "Дети", Description: "Детский сад, няня, кружки, одежда и игрушки"},
		{Name: "Питомцы", Description: "Корм, ветеринар, груминг"},
		{Name: "Ремонт и техника", Description: "Ремонт дома, замена бытовой техники", Rollover: true},
		{Name: "Авто: ремонт и ТО", Description: "Обслуживание и ремонт машины", Rollover: true},
		{Name: "Непредвиденные расходы", Description: "То, что случится, но неизвестно когда", Rollover: true},
		{Name: "Сбережения", Description: "Подушка безопасности и накопления", Rollover: true},
		{Name: "Отпуск", Description: "Накопления на поездку", Rollover: true},
	}
}

// ExportHeader — заголовок CSV-выгрузки записей, русский язык.
func ExportHeader() []string {
	return []string{"Дата", "Счёт", "Тип", "Категория", "Сумма", "Валюта", "Заметка"}
}

// KindName — тип записи для выгрузки, русский язык.
func KindName(k models.TransactionKind) string {
	switch k {
	case models.KindExpense:
		return "Расход"
	case models.KindIncome:
		return "Доход"
	case models.KindTransfer:
		return "Перевод"
	case models.KindExchange:
		return "Обмен"
	case models.KindFee:
		return "Комиссия"
	case models.KindOpening:
		return "Начальный остаток"
	}
	return string(k)
}

// DailyReminderText — ежедневное напоминание внести траты, русский язык.
func DailyReminderText() string {
	return "Вы внесли траты за сегодня? Откройте приложение и добавьте записи, пока ничего не забылось."
}

// LimitAlertText собирает предупреждение о лимите категории. Суммы в чат-уведомлениях не маскируются.
func LimitAlertText(a models.LimitAlert) string {
	amounts := money.Format(a.SpentMinor, a.Currency) + " из " + money.Format(a.BudgetedMinor, a.Currency)
	if a.Threshold >= models.LimitFull {
		return fmt.Sprintf("Лимит категории «%s» исчерпан: потрачено %s.", a.CategoryName, amounts)
	}
	return fmt.Sprintf("Категория «%s»: потрачено уже %d%% заложенного, %s.", a.CategoryName, models.LimitWarn, amounts)
}

// ReminderText собирает текст напоминания о платеже. Суммы в чат-уведомлениях не маскируются.
func ReminderText(r models.Reminder) string {
	s := r.Schedule
	amount := money.Format(s.AmountMinor, s.Currency)
	switch s.AmountKind {
	case models.AmountApprox:
		amount = "около " + amount
	case models.AmountRange:
		if s.AmountMaxMinor != nil {
			amount = "от " + amount + " до " + money.Format(*s.AmountMaxMinor, s.Currency)
		}
	}
	when := "Сегодня платёж"
	switch r.Lead {
	case 0:
	case 1:
		when = "Завтра платёж"
	default:
		when = fmt.Sprintf("Через %d дн. платёж", r.Lead)
	}
	return fmt.Sprintf("%s «%s»: %s.\nКогда оплатите, внесите трату в категорию платежа: он отметится оплаченным. Если платить не нужно, пропустите его в приложении.", when, s.Name, amount)
}

// Тексты бота для записи трат прямо в чате, русский язык. Суммы в чате не маскируются.

func BotHelp() string {
	return "Пришлите трату текстом: «кофе 5» или «такси 12 EUR». Доход начните с плюса: «+1000 зарплата».\nЕсли такая покупка уже была, категорию подберу сам, иначе предложу выбрать."
}

func BotUnknownUser() string {
	return "Сначала откройте приложение и пройдите настройку: после неё можно вносить траты сообщениями."
}

func BotNoAmount() string {
	return "Не нашёл сумму. Пример: «кофе 5» или «такси 12 EUR»."
}

func BotCurrencyMismatch() string {
	return "В этой валюте записать нельзя: у основного счёта другая валюта. Внесите такую запись в приложении."
}

func BotFailed() string {
	return "Не получилось записать, попробуйте ещё раз или внесите запись в приложении."
}

func BotPick(amount string, note string) string {
	if note == "" {
		return "Сумма " + amount + ". В какую категорию записать?"
	}
	return "Сумма " + amount + " («" + note + "»). В какую категорию записать?"
}

func BotSavedExpense(amount, category, note string) string {
	s := "Записал: " + amount + " · " + category
	if note != "" {
		s += " · " + note
	}
	return s
}

func BotSavedIncome(amount, note string) string {
	s := "Записал доход: " + amount
	if note != "" {
		s += " · " + note
	}
	return s
}

func BotUndoButton() string { return "Отменить" }

func BotUndone() string { return "Запись отменена." }

func BotExpired() string {
	return "Этот выбор устарел, пришлите трату ещё раз."
}
