package models

import "time"

// QualificationSession represents a scoring session for an event
type QualificationSession struct {
	UUID         string     `json:"id" db:"uuid"`
	EventUUID    string     `json:"event_id" db:"event_uuid"`
	SessionCode  string     `json:"session_code" db:"session_code"`
	SessionDate  *string    `json:"session_date" db:"session_date"`
	Name         string     `json:"name" db:"name"`
	StartTime    *time.Time `json:"start_time" db:"start_time"`
	EndTime      *time.Time `json:"end_time" db:"end_time"`
	TotalEnds    int        `json:"total_ends" db:"total_ends"`
	ArrowsPerEnd int        `json:"arrows_per_end" db:"arrows_per_end"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
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
	UUID            string    `json:"id" db:"uuid"`
	SessionUUID     string    `json:"session_id" db:"session_uuid"`
	ParticipantUUID string    `json:"participant_id" db:"participant_uuid"`
	EndNumber       int       `json:"end_number" db:"end_number"`
	TotalScoreEnd   int       `json:"total_score_end" db:"total_score_end"`
	XCountEnd       int       `json:"x_count_end" db:"x_count_end"`
	TenCountEnd     int       `json:"ten_count_end" db:"ten_count_end"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// ScoreUpdateRequest is the request payload for updating end scores
type ScoreUpdateRequest struct {
	Arrows    []string `json:"arrows" binding:"required"`
	EndNumber int      `json:"end_number" binding:"required"`
}
