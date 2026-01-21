package models

import "time"

// EventCategory represents a specific category for an event
type EventCategory struct {
	UUID         string    `json:"id" db:"uuid"`
	EventID      string    `json:"event_id" db:"event_id"`
	DivisionUUID string    `json:"division_id" db:"division_uuid"`
	CategoryUUID string    `json:"category_id" db:"category_uuid"`
	MaxParticipants *int   `json:"max_participants" db:"max_participants"`
	Status       string    `json:"status" db:"status"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
