package dialog

import (
	"sync"
	"time"
)

type State string

const (
	StateIdle        State = "idle"
	StateAskDistrict State = "ask_district"
	StateAskChildAge State = "ask_child_age"
	StateAskSport    State = "ask_sport"
	StateShowResults State = "show_results"
	StateCompleted   State = "completed"
	minChildAge            = 6
	maxChildAge            = 14
)

type UserState struct {
	UserID           int64
	CurrentState     State
	District         string
	ChildAge         int
	Sport            string
	SearchInProgress bool
	LastActivity     time.Time
}

func (state UserState) IsCompleted() bool {
	return state.CurrentState == StateCompleted
}

func (state UserState) IsIdle() bool {
	return state.CurrentState == StateIdle
}

func (state UserState) IsActionable() bool {
	switch state.CurrentState {
	case StateCompleted, StateShowResults:
		return false
	default:
		return true
	}
}

func (state UserState) HasDistrict() bool {
	return state.District != ""
}

func (state UserState) HasChildAge() bool {
	return state.ChildAge > 0
}

type Store struct {
	mu     sync.RWMutex
	states map[int64]UserState
}

func NewStore() *Store {
	return &Store{
		states: make(map[int64]UserState),
	}
}

func (store *Store) GetOrCreate(userID int64) UserState {
	if !isValidUserID(userID) {
		return UserState{
			UserID:       userID,
			CurrentState: StateIdle,
			LastActivity: time.Now(),
		}
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	if existingState, exists := store.states[userID]; exists {
		existingState.LastActivity = time.Now()
		store.states[userID] = existingState
		return existingState
	}

	newState := UserState{
		UserID:       userID,
		CurrentState: StateIdle,
		LastActivity: time.Now(),
	}
	store.states[userID] = newState
	return newState
}

func (store *Store) Get(userID int64) (UserState, bool) {
	if !isValidUserID(userID) {
		return UserState{}, false
	}

	store.mu.RLock()
	defer store.mu.RUnlock()

	state, exists := store.states[userID]
	return state, exists
}

func (store *Store) Set(state UserState) {
	if !isValidUserID(state.UserID) {
		return
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	state.LastActivity = time.Now()
	store.states[state.UserID] = state
}
func (store *Store) Reset(userID int64) {
	if !isValidUserID(userID) {
		return
	}

	resetState := UserState{
		UserID:       userID,
		CurrentState: StateIdle,
		LastActivity: time.Now(),
	}
	store.Set(resetState)
}
func (store *Store) Delete(userID int64) {
	if !isValidUserID(userID) {
		return
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	delete(store.states, userID)
}
func (store *Store) Count() int {
	store.mu.RLock()
	defer store.mu.RUnlock()

	return len(store.states)
}
func isValidChildAge(age int) bool {
	return age >= minChildAge && age <= maxChildAge
}

// isValidUserID проверяет, что идентификатор пользователя корректен.
// MAX не присылает нулевые userID; если такое произошло — это баг парсинга,
// и мы не должны писать состояние под ключом 0.
func isValidUserID(userID int64) bool {
	return userID > 0
}
func (state UserState) IsSearching() bool {
	return state.SearchInProgress
}

// HasSport возвращает true, если вид спорта уже выбран.
func (state UserState) HasSport() bool {
	return state.Sport != ""
}
