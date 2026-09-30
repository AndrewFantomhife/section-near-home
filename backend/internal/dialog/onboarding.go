package dialog

import (
	"context"
	"strconv"

	"section-near-home/backend/internal/clients/maxapi"
)

// Специальные значения payload спорта.
const (
	// sportValueUnknown запускает сценарий подсказок по ТЗ.
	sportValueUnknown = "unknown"

	// sportValueAny означает отказ от фильтра по спорту:
	// показываются все секции района, подходящие по возрасту.
	sportValueAny = "any"
)

// Количество кнопок спорта в одном ряду клавиатуры.
const sportButtonsPerRow = 2

// sportOption связывает отображаемое название кнопки с ключом секции в db.json.
type sportOption struct {
	DisplayName string
	CatalogKey  string
}

// sportOptions является каталогом видов спорта для клавиатуры выбора.
// Ровно 10 позиций согласно требованию ТЗ о списке не менее 10 видов спорта.
// CatalogKey обязан совпадать со значением поля sport в mock-api/db.json.
var sportOptions = []sportOption{
	{DisplayName: "Плавание", CatalogKey: "swimming"},
	{DisplayName: "Футбол", CatalogKey: "football"},
	{DisplayName: "Баскетбол", CatalogKey: "basketball"},
	{DisplayName: "Волейбол", CatalogKey: "volleyball"},
	{DisplayName: "Хоккей", CatalogKey: "hockey"},
	{DisplayName: "Теннис", CatalogKey: "tennis"},
	{DisplayName: "Гимнастика", CatalogKey: "gymnastics"},
	{DisplayName: "Бокс", CatalogKey: "boxing"},
	{DisplayName: "Карате", CatalogKey: "karate"},
	{DisplayName: "Лёгкая атлетика", CatalogKey: "athletics"},
}

// suggestedSportKeys содержит ключи спорта для сценария «не знаю».
var suggestedSportKeys = []string{"swimming", "football", "athletics"}

// availableDistricts содержит районы пилотного запуска.
// Список обязан совпадать со значениями поля district в mock-api/db.json.
var availableDistricts = []string{"Ленинский", "Октябрьский", "Фрунзенский"}

func (machine *Machine) startOnboarding(ctx context.Context, userID int64) error {
	buttons := buildDistrictKeyboard()

	if err := machine.maxClient.SendMessageWithKeyboard(ctx, userID, messageWelcome, buttons); err != nil {
		return err
	}

	state := machine.store.GetOrCreate(userID)
	state.District = ""
	state.ChildAge = 0
	state.Sport = ""
	state.CurrentState = StateAskDistrict
	machine.store.Set(state)
	return nil
}

func buildDistrictKeyboard() [][]maxapi.Button {
	buttons := make([][]maxapi.Button, 0, len(availableDistricts))
	for _, district := range availableDistricts {
		buttons = append(buttons, []maxapi.Button{{
			Text:    district,
			Payload: payloadPrefixDistrict + district,
		}})
	}
	return buttons
}

// processDistrict валидирует выбранный район и переходит к шагу возраста.
// Район вне белого списка считается подделанным payload: клавиатура повторяется.
func (machine *Machine) processDistrict(ctx context.Context, userID int64, district string) error {
	if !isKnownDistrict(district) {
		machine.logger.WarnContext(ctx, "unknown district payload", "district", district)
		return machine.startOnboarding(ctx, userID)
	}

	state := machine.store.GetOrCreate(userID)
	state.District = district
	state.CurrentState = StateAskChildAge
	machine.store.Set(state)

	return machine.askChildAge(ctx, userID)
}

// askChildAge запрашивает возраст ребёнка клавиатурой в две строки: 6-10 и 11-14.
func (machine *Machine) askChildAge(ctx context.Context, userID int64) error {
	state := machine.store.GetOrCreate(userID)
	buttons := buildAgeKeyboard()

	if err := machine.maxClient.SendMessageWithKeyboard(
		ctx, userID, formatAskChildAge(state.District), buttons); err != nil {
		return err
	}

	state.CurrentState = StateAskChildAge
	machine.store.Set(state)
	return nil
}

// processChildAge сохраняет возраст и переходит к шагу выбора спорта.
// Диапазон уже проверен в machine.go при разборе payload.
func (machine *Machine) processChildAge(ctx context.Context, userID int64, age int) error {
	state := machine.store.GetOrCreate(userID)
	state.ChildAge = age
	machine.store.Set(state)

	return machine.askSport(ctx, userID)
}

// askSport запрашивает вид спорта клавиатурой из 10 кнопок и кнопки «не знаю».
func (machine *Machine) askSport(ctx context.Context, userID int64) error {
	buttons := buildSportKeyboard()

	if err := machine.maxClient.SendMessageWithKeyboard(ctx, userID, messageAskSport, buttons); err != nil {
		return err
	}

	state := machine.store.GetOrCreate(userID)
	state.CurrentState = StateAskSport
	machine.store.Set(state)
	return nil
}

// processSport сохраняет выбранный спорт и передаёт управление показу результатов.
// Специальные значения: unknown запускает подсказки, any отключает фильтр спорта.
func (machine *Machine) processSport(ctx context.Context, userID int64, sportKey string) error {
	switch sportKey {
	case sportValueUnknown:
		return machine.suggestSports(ctx, userID)
	case sportValueAny:
		// Фильтр по спорту не применяется.
	default:
		if !isKnownSportKey(sportKey) {
			machine.logger.WarnContext(ctx, "unknown sport payload", "sport", sportKey)
			return machine.suggestSports(ctx, userID)
		}
	}

	state := machine.store.GetOrCreate(userID)
	if sportKey == sportValueAny {
		state.Sport = ""
	} else {
		state.Sport = sportKey
	}
	machine.store.Set(state)

	return machine.showResults(ctx, userID)
}

// suggestSports реализует альтернативный сценарий «не знаю» по ТЗ:
// бот предлагает популярные варианты и кнопку показа всех секций без фильтра.
func (machine *Machine) suggestSports(ctx context.Context, userID int64) error {
	row := make([]maxapi.Button, 0, len(suggestedSportKeys))
	for _, key := range suggestedSportKeys {
		option, found := findSportOption(key)
		if !found {
			machine.logger.Warn("suggested sport missing in catalog", "sport", key)
			continue
		}
		row = append(row, maxapi.Button{
			Text:    option.DisplayName,
			Payload: payloadPrefixSport + option.CatalogKey,
		})
	}

	buttons := [][]maxapi.Button{
		row,
		{{Text: messageShowAllSportsButton, Payload: payloadPrefixSport + sportValueAny}},
	}

	return machine.maxClient.SendMessageWithKeyboard(ctx, userID, messageSuggestSport, buttons)
}

// isKnownDistrict возвращает true, если район присутствует в белом списке.
func isKnownDistrict(district string) bool {
	for _, knownDistrict := range availableDistricts {
		if knownDistrict == district {
			return true
		}
	}
	return false
}

// isKnownSportKey возвращает true, если ключ спорта присутствует в каталоге.
func isKnownSportKey(sportKey string) bool {
	_, found := findSportOption(sportKey)
	return found
}

// findSportOption ищет опцию каталога по ключу спорта.
func findSportOption(sportKey string) (sportOption, bool) {
	for _, option := range sportOptions {
		if option.CatalogKey == sportKey {
			return option, true
		}
	}
	return sportOption{}, false
}

// buildAgeKeyboard формирует клавиатуру возраста в две строки: 6-10 и 11-14.
func buildAgeKeyboard() [][]maxapi.Button {
	firstRow := make([]maxapi.Button, 0, 5)
	for age := minChildAge; age <= 10; age++ {
		firstRow = append(firstRow, maxapi.Button{
			Text:    strconv.Itoa(age),
			Payload: payloadPrefixAge + strconv.Itoa(age),
		})
	}

	secondRow := make([]maxapi.Button, 0, 4)
	for age := 11; age <= maxChildAge; age++ {
		secondRow = append(secondRow, maxapi.Button{
			Text:    strconv.Itoa(age),
			Payload: payloadPrefixAge + strconv.Itoa(age),
		})
	}

	return [][]maxapi.Button{firstRow, secondRow}
}

// buildSportKeyboard формирует клавиатуру спорта: 10 кнопок по две в ряду
// и завершающая кнопка «не знаю, предложите сами».
func buildSportKeyboard() [][]maxapi.Button {
	rows := make([][]maxapi.Button, 0, len(sportOptions)/sportButtonsPerRow+2)
	row := make([]maxapi.Button, 0, sportButtonsPerRow)

	for _, option := range sportOptions {
		row = append(row, maxapi.Button{
			Text:    option.DisplayName,
			Payload: payloadPrefixSport + option.CatalogKey,
		})
		if len(row) == sportButtonsPerRow {
			rows = append(rows, row)
			row = make([]maxapi.Button, 0, sportButtonsPerRow)
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}

	rows = append(rows, []maxapi.Button{{
		Text:    messageSuggestSportButton,
		Payload: payloadPrefixSport + sportValueUnknown,
	}})
	return rows
}
