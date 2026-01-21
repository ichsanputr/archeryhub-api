package models

import (
	"time"
)

// Archer represents an archer
type Archer struct {
	UUID             string     `json:"id" db:"uuid"`
	UserID           *string    `json:"user_id" db:"user_id"`
	ArcherCode       *string    `json:"archer_code" db:"athlete_code"`
	FullName         string     `json:"full_name" db:"full_name"`
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

// ArcherWithStats includes statistics
type ArcherWithStats struct {
	Archer
	TotalEvents     int        `json:"total_events" db:"total_events"`
	CompletedEvents int        `json:"completed_events" db:"completed_events"`
	LastEventDate   *time.Time `json:"last_event_date" db:"last_event_date"`
}

// CreateArcherRequest represents the request payload for creating an archer
type CreateArcherRequest struct {
	FullName         string     `json:"full_name" binding:"required,min=2,max=100"`
	ArcherCode       *string    `json:"archer_code"`
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

// UpdateArcherRequest represents the request payload for updating an archer
type UpdateArcherRequest struct {
	FullName         *string    `json:"full_name"`
	ArcherCode       *string    `json:"archer_code"`
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

// EventParticipant represents an archer registered for an event
type EventParticipant struct {
	UUID                string    `json:"id" db:"uuid"`
	EventID             string    `json:"event_id" db:"event_id"`
	ArcherID            string    `json:"archer_id" db:"archer_id"`
	CategoryID          string    `json:"category_id" db:"category_id"`
	BackNumber          *string   `json:"back_number" db:"back_number"`
	TargetNumber        *string   `json:"target_number" db:"target_number"`
	Session             *int      `json:"session" db:"session"`
	RegistrationDate    time.Time `json:"registration_date" db:"registration_date"`
	PaymentStatus       string    `json:"payment_status" db:"payment_status"` // pending, paid, waived, refunded
	PaymentAmount       float64   `json:"payment_amount" db:"payment_amount"`
	AccreditationStatus string    `json:"accreditation_status" db:"accreditation_status"` // pending, printed, collected
	Notes               *string   `json:"notes" db:"notes"`
}

// ParticipantWithDetails includes archer and event information
type ParticipantWithDetails struct {
	EventParticipant
	FullName     string  `json:"full_name" db:"full_name"`
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

// RegisterParticipantRequest represents the request to register an archer to an event
type RegisterParticipantRequest struct {
	ArcherID      string   `json:"archer_id" binding:"required"`
	CategoryID    string   `json:"category_id" binding:"required"`
	BackNumber    *string  `json:"back_number"`
	TargetNumber  *string  `json:"target_number"`
	Session       *int     `json:"session"`
	PaymentStatus *string  `json:"payment_status" binding:"omitempty,oneof=pending paid waived"`
	PaymentAmount *float64 `json:"payment_amount"`
	Notes         *string  `json:"notes"`
}

// BulkImportArcher represents an archer record for bulk import
type BulkImportArcher struct {
	FullName    string `json:"full_name"`
	DateOfBirth string `json:"date_of_birth"`
	Gender      string `json:"gender"`
	Country     string `json:"country"`
	Club        string `json:"club"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Division    string `json:"division"`
	Category    string `json:"category"`
}
