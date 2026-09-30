package dialog

import "fmt"

// Служебные сообщения маршрутизации и fallback-сценариев.
const (
	messageNonTextOnly = "Я понимаю только текст и кнопки. Используйте /start, чтобы начать."

	messageDistrictButtonsOnly = "Пожалуйста, выберите район кнопками ниже."

	messageAgeButtonsOnly = "Пожалуйста, выберите возраст ребёнка кнопками ниже."

	messageSportButtonsOnly = "Пожалуйста, выберите вид спорта кнопками ниже."

	messageSearchInProgress = "Подбор секций ещё идёт, пожалуйста, подождите несколько секунд."

	messageHelp = "Я помогу найти бесплатную или бюджетную спортивную секцию рядом с домом.\n\n" +
		"Доступные команды:\n" +
		"/start - начать диалог заново\n" +
		"/help - показать эту справку\n" +
		"/reset - сбросить текущий диалог"

	messageResetDone = "Диалог сброшен. Введите /start, чтобы начать заново."

	messageUnknownCommand = "Неизвестная команда. Доступны /start и /help."

	messageUnknownAction = "Неизвестное действие"

	messageInvalidAgeCallback = "Возраст должен быть от 6 до 14 лет"

	messageShowAllSportsButton = "Показать все виды спорта"
	messageSuggestSportButton  = "Не знаю, предложите"

	messageSearchFailed  = "Не удалось получить список секций. Попробуйте ещё раз через минуту."
	messageRestartButton = "Начать заново"
)

// Сообщения онбординга. Используются в onboarding.go.
const (
	messageWelcome = "Привет! Я помогу найти спортивную секцию рядом с домом.\n" +
		"Задам 3 вопроса, это займёт не больше 5 минут.\n\n" +
		"Нажмите кнопку ниже, чтобы начать."

	messageWelcomeButton = "Начать подбор"

	messageAskDistrict = "В каком районе ищете секцию?\nВыберите район кнопками ниже."

	messageAskSport = "Какой вид спорта интересует?\nВыберите кнопками ниже."

	messageSuggestSport = "Ничего страшного! Вот что чаще всего выбирают дети 6-14 лет.\n" +
		"Выберите вариант кнопками или посмотрите все секции без фильтра по спорту."
)

// Сообщения результатов. Используются в results.go.
const (
	messageSearching = "Ищу подходящие секции рядом... это займёт несколько секунд."

	messageNoResults = "По вашему запросу секций не найдено.\n\n" +
		"Попробуйте изменить параметры командой /start."

	messageResultsFooter = "Чтобы сравнить все секции района, откройте карту.\n" +
		"Для нового поиска введите /start."

	messageOpenMapButton = "Открыть карту"
)

// formatAskChildAge формирует вопрос о возрасте с подтверждением района.
func formatAskChildAge(district string) string {
	return fmt.Sprintf("Принял: %s.\nСколько лет ребёнку? Выберите кнопками ниже.", district)
}

// formatResultsHeader формирует заголовок списка найденных секций.
func formatResultsHeader(count int) string {
	return fmt.Sprintf("Нашёл %d подходящих секций:", count)
}
