// Package i18n хранит тексты, которые бэкенд сам кладёт в данные пользователя
// (стартовые категории). Тексты интерфейса живут в каталоге фронта.
package i18n

import "moneykeeper/internal/models"

// DefaultGroups — стартовые группы категорий в порядке показа, русский язык.
func DefaultGroups() []models.DefaultGroup {
	return []models.DefaultGroup{
		{Key: models.GroupCritical, Name: "Критичные"},
		{Key: models.GroupDebt, Name: "Долги"},
		{Key: models.GroupDaily, Name: "Ежедневные"},
		{Key: models.GroupRainy, Name: "На чёрный день"},
		{Key: models.GroupGoal, Name: "Цели накопления"},
	}
}

// DefaultCategories — стартовый набор категорий по группам из README, русский язык.
func DefaultCategories() []models.DefaultCategory {
	return []models.DefaultCategory{
		{Group: models.GroupCritical, Name: "Аренда и ипотека", Description: "Жильё: аренда или платёж по ипотеке"},
		{Group: models.GroupCritical, Name: "Коммунальные услуги", Description: "Свет, вода, газ, отопление"},
		{Group: models.GroupCritical, Name: "Налоги", Description: "Обязательные налоги и сборы"},
		{Group: models.GroupDebt, Name: "Погашение долгов", Description: "Платежи по кредитам, картам и займам"},
		{Group: models.GroupDaily, Name: "Продукты", Description: "Еда и товары для дома"},
		{Group: models.GroupDaily, Name: "Транспорт и бензин", Description: "Проезд, такси, топливо"},
		{Group: models.GroupDaily, Name: "Питание вне дома", Description: "Кафе, рестораны, кофе, доставка"},
		{Group: models.GroupDaily, Name: "Одежда", Description: "Одежда и обувь"},
		{Group: models.GroupDaily, Name: "Развлечения", Description: "Кино, игры, хобби, отдых"},
		{Group: models.GroupDaily, Name: "Подарки", Description: "Подарки близким и на праздники"},
		{Group: models.GroupDaily, Name: "Личная гигиена", Description: "Уход, косметика, парикмахерская"},
		{Group: models.GroupDaily, Name: "Подписки", Description: "Интернет, связь, сервисы по подписке"},
		{Group: models.GroupDaily, Name: "Благотворительность", Description: "Пожертвования и помощь"},
		{Group: models.GroupDaily, Name: "Уход за детьми", Description: "Детский сад, няня, кружки"},
		{Group: models.GroupGoal, Name: "Сбережения", Description: "Подушка безопасности и накопления"},
	}
}
