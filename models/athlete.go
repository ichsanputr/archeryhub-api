package models

import (
	"time"
)

// Athlete represents an archer/athlete
type Athlete struct {
	ID               string     `json:"id" db:"id"`
	UserID           *string    `json:"user_id" db:"user_id"`
	AthleteCode      *string    `json:"athlete_code" db:"athlete_code"`
	FirstName        string     `json:"first_name" db:"first_name"`
	LastName         string     `json:"last_name" db:"last_name"`
	DateOfBirth      *time.Time `json:"date_of_birth" db:"date_of_birth"`
	Gender           *string    `json:"gender" db:"gender"` // M, F, X
	Country          *string    `json:"country" db:"country"`
	Club             *string    `json:"club" db:"club"`
	Email            *string    `json:"email" db:"email"`
	Phone            *string    `json:"phone" db:"phone"`
	PhotoURL         *string    `json:"photo_url" db:"photo_url"`
	Address          *string    `json:"address" db:"address"`
	EmergencyContact *string    `json:"emergency_contact" db:"emergency_contact"`
	EmergencyPhone   *string    `json:"emergency_phone" db:"emergency_phone"`
	Status           string     `json:"status" db:"status"` // active, inactive, suspended, pending
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// AthleteWithStats includes statistics
type AthleteWithStats struct {
	Athlete
	TotalEvents     int        `json:"total_events" db:"total_events"`
	CompletedEvents int        `json:"completed_events" db:"completed_events"`
	LastEventDate   *time.Time `json:"last_event_date" db:"last_event_date"`
}

// CreateAthleteRequest represents the request payload for creating an athlete
type CreateAthleteRequest struct {
	FirstName        string     `json:"first_name" binding:"required,min=2,max=50"`
	LastName         string     `json:"last_name" binding:"required,min=2,max=50"`
	AthleteCode      *string    `json:"athlete_code"`
	DateOfBirth      *time.Time `json:"date_of_birth"`
	Gender           *string    `json:"gender" binding:"omitempty,oneof=M F X"`
	Country          *string    `json:"country"`
	Club             *string    `json:"club"`
	Email            *string    `json:"email" binding:"omitempty,email"`
	Phone            *string    `json:"phone"`
	PhotoURL         *string    `json:"photo_url"`
	Address          *string    `json:"address"`
	EmergencyContact *string    `json:"emergency_contact"`
	EmergencyPhone   *string    `json:"emergency_phone"`
}

// UpdateAthleteRequest represents the request payload for updating an athlete
type UpdateAthleteRequest struct {
	FirstName        *string    `json:"first_name"`
	LastName         *string    `json:"last_name"`
	AthleteCode      *string    `json:"athlete_code"`
	DateOfBirth      *time.Time `json:"date_of_birth"`
	Gender           *string    `json:"gender" binding:"omitempty,oneof=M F X"`
	Country          *string    `json:"country"`
	Club             *string    `json:"club"`
	Email            *string    `json:"email" binding:"omitempty,email"`
	Phone            *string    `json:"phone"`
	PhotoURL         *string    `json:"photo_url"`
	Address          *string    `json:"address"`
	EmergencyContact *string    `json:"emergency_contact"`
	EmergencyPhone   *string    `json:"emergency_phone"`
	Status           *string    `json:"status" binding:"omitempty,oneof=active inactive suspended pending"`
}

// TournamentParticipant represents an athlete registered for a tournament event
type TournamentParticipant struct {
	ID                  string    `json:"id" db:"id"`
	TournamentID        string    `json:"tournament_id" db:"tournament_id"`
	AthleteID           string    `json:"athlete_id" db:"athlete_id"`
	EventID             string    `json:"event_id" db:"event_id"`
	BackNumber          *string   `json:"back_number" db:"back_number"`
	TargetNumber        *string   `json:"target_number" db:"target_number"`
	Session             *int      `json:"session" db:"session"`
	RegistrationDate    time.Time `json:"registration_date" db:"registration_date"`
	PaymentStatus       string    `json:"payment_status" db:"payment_status"` // pending, paid, waived, refunded
	PaymentAmount       float64   `json:"payment_amount" db:"payment_amount"`
	AccreditationStatus string    `json:"accreditation_status" db:"accreditation_status"` // pending, printed, collected
	Notes               *string   `json:"notes" db:"notes"`
}

// ParticipantWithDetails includes athlete and event information
type ParticipantWithDetails struct {
	TournamentParticipant
	FirstName    string  `json:"first_name" db:"first_name"`
	LastName     string  `json:"last_name" db:"last_name"`
	Country      *string `json:"country" db:"country"`
	Club         *string `json:"club" db:"club"`
	PhotoURL     *string `json:"photo_url" db:"photo_url"`
	DivisionName string  `json:"division_name" db:"division_name"`
	DivisionCode string  `json:"division_code" db:"division_code"`
	CategoryName string  `json:"category_name" db:"category_name"`
	CategoryCode string  `json:"category_code" db:"category_code"`
	QualScore    *int    `json:"qual_score" db:"qual_score"`
	QualRank     *int    `json:"qual_rank" db:"qual_rank"`
}

// RegisterParticipantRequest represents the request to register an athlete to a tournament
type RegisterParticipantRequest struct {
	AthleteID     string   `json:"athlete_id" binding:"required"`
	EventID       string   `json:"event_id" binding:"required"`
	BackNumber    *string  `json:"back_number"`
	TargetNumber  *string  `json:"target_number"`
	Session       *int     `json:"session"`
	PaymentStatus *string  `json:"payment_status" binding:"omitempty,oneof=pending paid waived"`
	PaymentAmount *float64 `json:"payment_amount"`
	Notes         *string  `json:"notes"`
}

// BulkImportAthlete represents an athlete record for bulk import
type BulkImportAthlete struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DateOfBirth string `json:"date_of_birth"`
	Gender      string `json:"gender"`
	Country     string `json:"country"`
	Club        string `json:"club"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Division    string `json:"division"`
	Category    string `json:"category"`
}
