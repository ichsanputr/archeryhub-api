package models

import "time"

// EventCategory represents a specific category for an event
type EventCategory struct {
	UUID               string    `json:"id" db:"uuid"`
	EventID            string    `json:"event_id" db:"tournament_id"`
	DivisionUUID       string    `json:"division_id" db:"division_uuid"`
	CategoryUUID       string    `json:"category_id" db:"category_uuid"`
	EventTypeUUID      string    `json:"event_type_id" db:"tournament_type_uuid"`
	GenderDivisionUUID *string   `json:"gender_division_id" db:"gender_division_uuid"`
	Status             string    `json:"status" db:"status"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}
