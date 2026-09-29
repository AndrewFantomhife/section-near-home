package domain

import "time"

// WaitlistEntry описывает запись в очереди на освободившееся место в секции.
// Родитель попадает в очередь, когда свободных мест в секции нет.
type WaitlistEntry struct {
	ID        int       `json:"id"`
	SectionID int       `json:"sectionId"`
	UserID    int64     `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
}
