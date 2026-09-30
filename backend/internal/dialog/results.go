package dialog

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"section-near-home/backend/internal/clients/maxapi"
	"section-near-home/backend/internal/domain"
	"section-near-home/backend/internal/logging"
)

// maxResultCards ограничивает число карточек секций в одном сообщении.
const maxResultCards = 3

// cardSeparator визуально отделяет карточки секций внутри одного сообщения.
const cardSeparator = "\n----------\n"

// showResults выполняет подбор секций и отправляет результат пользователю.
//
// Контракт состояний:
//   - на время подбора поднимается флаг SearchInProgress, состояние остаётся
//     StateAskSport; любые входящие сообщения получают честный ответ о идущем
//     подборе, а нажатия кнопок игнорируются, что исключает ложные обещания
//     и повторный подбор при двойном нажатии;
//   - переход в StateCompleted выполняется только после успешной отправки
//     карточек либо завершающего сообщения без карточек;
//   - при ошибке отправки карточек отправляется fallback-сообщение с кнопкой
//     повтора; если и оно не дошло, состояние остаётся StateAskSport,
//     и пользователь может повторить подбор нажатием кнопки спорта.
func (machine *Machine) showResults(ctx context.Context, userID int64) error {
	state := machine.store.GetOrCreate(userID)
	state.SearchInProgress = true
	machine.store.Set(state)

	defer func() {
		current := machine.store.GetOrCreate(userID)
		current.SearchInProgress = false
		machine.store.Set(current)
	}()

	// Статус по ТЗ: текстовое сообщение уходит до запроса к справочнику,
	// поэтому появляется не позднее одной секунды после ответа онбординга.
	if err := machine.maxClient.SendMessage(ctx, userID, messageSearching); err != nil {
		return err
	}

	// Индикатор набора текста покрывает время ожидания ответа справочника.
	if err := machine.maxClient.SendTyping(ctx, userID); err != nil {
		machine.logger.WarnContext(ctx, "send typing failed", "error", err)
	}

	sections, err := machine.mockClient.ListSections(ctx)
	if err != nil {
		logging.ErrorWithCause(machine.logger, "list sections failed", err)
		return machine.finishResults(ctx, userID, messageSearchFailed)
	}

	filtered := filterSections(sections, state)
	machine.logger.InfoContext(ctx, "sections filtered",
		"district", state.District,
		"child_age", state.ChildAge,
		"sport", state.Sport,
		"total", len(sections),
		"matched", len(filtered))

	if len(filtered) == 0 {
		return machine.finishResults(ctx, userID, messageNoResults)
	}

	sortSections(filtered)
	if len(filtered) > maxResultCards {
		filtered = filtered[:maxResultCards]
	}

	buttons := machine.buildResultsKeyboard(state)
	if err := machine.maxClient.SendMessageWithKeyboard(
		ctx, userID, buildResultsMessage(filtered), buttons); err != nil {
		logging.ErrorWithCause(machine.logger, "send results failed", err)
		return machine.finishResults(ctx, userID, messageSearchFailed)
	}

	completed := machine.store.GetOrCreate(userID)
	completed.CurrentState = StateCompleted
	machine.store.Set(completed)
	return nil
}

// finishResults завершает диалог сообщением без карточек и кнопкой повтора.
// Переход в StateCompleted выполняется только после успешной отправки,
// поэтому пользователь никогда не остаётся в терминальном состоянии
// без полученного сообщения.
func (machine *Machine) finishResults(ctx context.Context, userID int64, text string) error {
	buttons := [][]maxapi.Button{{
		{Text: messageRestartButton, Payload: payloadActionRestart},
	}}

	if err := machine.maxClient.SendMessageWithKeyboard(ctx, userID, text, buttons); err != nil {
		return err
	}

	completed := machine.store.GetOrCreate(userID)
	completed.CurrentState = StateCompleted
	machine.store.Set(completed)
	return nil
}

// buildResultsKeyboard формирует клавиатуру результата с кнопкой перехода
// в мини-приложение. Вместо open_app используется кнопка-ссылка: она
// открывает указанный URL напрямую и не требует регистрации мини-приложения
// в настройках бота MAX.
//
// Контекст подбора (вид спорта, возраст) передаётся в мини-приложение
// через query-параметры — фронт использует их для предзаполнения фильтра.
func (machine *Machine) buildResultsKeyboard(state UserState) [][]maxapi.Button {
	url := fmt.Sprintf("http://localhost:8081/?type=%s&age=%d",
		state.Sport, state.ChildAge)

	return [][]maxapi.Button{{
		{Text: messageOpenMapButton, LinkURL: url},
	}}
}

// filterSections применяет параметры пользователя к списку секций.
// Район сравнивается точно: значение приходит из кнопок белого списка,
// поэтому опечатки пользователя невозможны, а нестрогое совпадение
// создавало бы ложные срабатывания на подстроках.
// Спорт сравнивается точным совпадением ключа каталога.
// Возраст проверяется по нижней границе секции.
func filterSections(sections []domain.Section, state UserState) []domain.Section {
	result := make([]domain.Section, 0, len(sections))

	for _, section := range sections {
		if state.HasChildAge() && !section.FitsAge(state.ChildAge) {
			continue
		}
		if state.HasSport() && section.Sport != state.Sport {
			continue
		}
		if state.HasDistrict() && section.District != state.District {
			continue
		}
		result = append(result, section)
	}

	return result
}

// sortSections упорядочивает секции по полезности для родителя:
// сначала со свободными местами, затем бесплатные, затем по идентификатору
// для детерминированности выдачи.
func sortSections(sections []domain.Section) {
	sort.SliceStable(sections, func(leftIndex, rightIndex int) bool {
		left := sections[leftIndex]
		right := sections[rightIndex]
		if left.HasFreeSpots() != right.HasFreeSpots() {
			return left.HasFreeSpots()
		}
		if left.IsFree() != right.IsFree() {
			return left.IsFree()
		}
		return left.ID < right.ID
	})
}

// buildResultsMessage собирает итоговое сообщение: заголовок, блок карточек, подвал.
func buildResultsMessage(sections []domain.Section) string {
	var builder strings.Builder
	builder.WriteString(formatResultsHeader(len(sections)))
	builder.WriteString("\n\n")
	builder.WriteString(joinCards(sections))
	builder.WriteString("\n\n")
	builder.WriteString(messageResultsFooter)
	return builder.String()
}

// joinCards склеивает карточки секций в один текстовый блок.
func joinCards(sections []domain.Section) string {
	cards := make([]string, 0, len(sections))
	for _, section := range sections {
		cards = append(cards, formatCard(section))
	}
	return strings.Join(cards, cardSeparator)
}

// formatCard формирует человекочитаемую карточку секции для чата.
func formatCard(section domain.Section) string {
	var builder strings.Builder
	builder.WriteString(section.Name)
	builder.WriteString("\nСпорт: ")
	builder.WriteString(sportDisplayName(section.Sport))
	builder.WriteString("\nАдрес: ")
	builder.WriteString(section.Address)
	builder.WriteString(" (")
	builder.WriteString(section.District)
	builder.WriteString(")\nРасписание: ")
	builder.WriteString(section.Schedule)
	builder.WriteString("\nВозраст: от ")
	builder.WriteString(fmt.Sprintf("%d", section.MinAge))
	builder.WriteString(" лет")
	builder.WriteString("\nСтоимость: ")
	builder.WriteString(section.FormatPrice())
	builder.WriteString("\nМеста: ")
	builder.WriteString(formatSpots(section))
	return builder.String()
}

// formatSpots возвращает человекочитаемое описание свободных мест.
func formatSpots(section domain.Section) string {
	if !section.HasFreeSpots() {
		return "мест нет"
	}
	return fmt.Sprintf("%d свободных", section.Spots)
}

// sportDisplayName возвращает отображаемое название спорта по ключу каталога.
// Для неизвестного ключа возвращает сам ключ, чтобы не терять информацию.
func sportDisplayName(catalogKey string) string {
	option, found := findSportOption(catalogKey)
	if !found {
		return catalogKey
	}
	return option.DisplayName
}
