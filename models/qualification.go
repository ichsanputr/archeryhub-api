package models

import "time"

// QualificationSession represents a scoring session for a specific category
type QualificationSession struct {
	UUID              string    `json:"id" db:"uuid"`
	EventCategoryUUID string    `json:"event_category_id" db:"event_category_uuid"`
	SessionName       string    `json:"session_name" db:"session_name"`
	SessionOrder      int       `json:"session_order" db:"session_order"`
	StartTime         *time.Time `json:"start_time" db:"start_time"`
	Status            string    `json:"status" db:"status"` // draft, ongoing, completed
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// QualificationAssignment maps a participant to a target in a session
type QualificationAssignment struct {
	UUID            string    `json:"id" db:"uuid"`
	SessionUUID     string    `json:"session_id" db:"session_uuid"`
	ParticipantUUID string    `json:"participant_id" db:"participant_uuid"`
	TargetNumber    int       `json:"target_number" db:"target_number"`
	TargetPosition  string    `json:"target_position" db:"target_position"` // A, B, C, D
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// QualificationEndScore represents the score for a single end (set of arrows)
type QualificationEndScore struct {
	UUID           string    `json:"id" db:"uuid"`
	AssignmentUUID string    `json:"assignment_id" db:"assignment_uuid"`
	EndNumber      int       `json:"end_number" db:"end_number"`
	Arrow1         string    `json:"arrow_1" db:"arrow_1"` // X, 10, 9, ..., M, null
	Arrow2         string    `json:"arrow_2" db:"arrow_2"`
	Arrow3         string    `json:"arrow_3" db:"arrow_3"`
	Arrow4         string    `json:"arrow_4" db:"arrow_4"`
	Arrow5         string    `json:"arrow_5" db:"arrow_5"`
	Arrow6         string    `json:"arrow_6" db:"arrow_6"`
	EndTotal       int       `json:"end_total" db:"end_total"`
	EndXCount      int       `json:"end_x_count" db:"end_x_count"`
	End10Count     int       `json:"end_10_count" db:"end_10_count"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// ScoreUpdateRequest is the request payload for updating end scores
type ScoreUpdateRequest struct {
	Arrows    []string `json:"arrows" binding:"required"`
	EndNumber int      `json:"end_number" binding:"required"`
}
