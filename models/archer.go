package models

import (
	"time"
)

// Archer represents an archer
type Archer struct {
	UUID         string     `json:"id" db:"uuid"`
	UserID       *string    `json:"user_id" db:"user_id"`
	Username     *string    `json:"username" db:"username"`
	FullName     string     `json:"full_name" db:"full_name"`
	DateOfBirth  *time.Time `json:"date_of_birth" db:"date_of_birth"`
	Gender       *string    `json:"gender" db:"gender"` // M, F, X
	Club         *string    `json:"club" db:"club"`
	Email        *string    `json:"email" db:"email"`
	Phone        *string    `json:"phone" db:"phone"`
	AvatarURL    *string    `json:"avatar_url" db:"avatar_url"`
	Address      *string    `json:"address" db:"address"`
	Bio          *string    `json:"bio" db:"bio"`
	Achievements *string    `json:"achievements" db:"achievements"`
	Status       string     `json:"status" db:"status"` // active, inactive, suspended, pending
	IsVerified   bool       `json:"is_verified" db:"is_verified"`
	BowType      *string    `json:"bow_type" db:"bow_type"`
	City         *string    `json:"city" db:"city"`
	School       *string    `json:"school" db:"school"`
	Province     *string    `json:"province" db:"province"`
	CustomID     string     `json:"custom_id" db:"custom_id"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// ArcherWithStats includes statistics
type ArcherWithStats struct {
	Archer
	ClubName        *string    `json:"club_name" db:"club_name"`
	ClubSlug        *string    `json:"club_slug" db:"club_slug"`
	TotalEvents     int        `json:"total_events" db:"total_events"`
	CompletedEvents int        `json:"completed_events" db:"completed_events"`
	LastEventDate   *time.Time `json:"last_event_date" db:"last_event_date"`
}

// CreateArcherRequest represents the request payload for creating an archer
type CreateArcherRequest struct {
	FullName    string     `json:"full_name" binding:"required,min=2,max=100"`
	Username    *string    `json:"username"`
	Email       *string    `json:"email" binding:"omitempty,email"`
	Password    *string    `json:"password"`
	Nickname    *string       `json:"nickname"`
	DateOfBirth *FlexibleTime `json:"date_of_birth"`
	Gender      *string       `json:"gender" binding:"omitempty,oneof=male female M F"`
	BowType     *string    `json:"bow_type" binding:"omitempty,oneof=recurve compound barebow traditional"`
	City        *string    `json:"city"`
	School      *string    `json:"school"`
	Club        *string    `json:"club"`
	ClubID      *string    `json:"club_id"`
	Phone       *string    `json:"phone"`
	AvatarURL   *string    `json:"avatar_url"`
	Address     *string    `json:"address"`
}

// UpdateArcherRequest represents the request payload for updating an archer
type UpdateArcherRequest struct {
	FullName     *string       `json:"full_name"`
	DateOfBirth  *FlexibleTime `json:"date_of_birth"`
	Gender       *string       `json:"gender" binding:"omitempty,oneof=M F X"`
	Club         *string    `json:"club"`
	City         *string    `json:"city"`
	School       *string    `json:"school"`
	Email        *string    `json:"email" binding:"omitempty,email"`
	Phone        *string    `json:"phone"`
	AvatarURL    *string    `json:"avatar_url"`
	Address      *string    `json:"address"`
	Bio          *string    `json:"bio"`
	Achievements *string    `json:"achievements"`
	Status       *string    `json:"status" binding:"omitempty,oneof=active inactive suspended pending"`
}

// EventParticipant represents an archer registered for an event
type EventParticipant struct {
	UUID                string    `json:"id" db:"uuid"`
	EventID             string    `json:"event_id" db:"event_id"`
	ArcherID            string    `json:"archer_id" db:"archer_id"`
	EventArcherID       *string   `json:"event_archer_id" db:"event_archer_id"`
	CategoryID          string    `json:"category_id" db:"category_id"`
	BackNumber          *string   `json:"back_number" db:"back_number"`
	TargetNumber        *string   `json:"target_number" db:"target_number"`
	Session             *int      `json:"session" db:"session"`
	RegistrationDate    time.Time `json:"registration_date" db:"registration_date"`
	PaymentStatus       string    `json:"payment_status" db:"payment_status"` // menunggu_acc, belum_lunas, lunas
	PaymentAmount       float64   `json:"payment_amount" db:"payment_amount"`
	AccreditationStatus string    `json:"accreditation_status" db:"accreditation_status"` // pending, printed, collected
	Notes               *string   `json:"notes" db:"notes"`
}

// ParticipantWithDetails includes archer and event information
type ParticipantWithDetails struct {
	EventParticipant
	FullName     string  `json:"full_name" db:"full_name"`
	City         *string `json:"city" db:"city"`
	Club         *string `json:"club" db:"club"`
	AvatarURL    *string `json:"avatar_url" db:"avatar_url"`
	DivisionName string  `json:"division_name" db:"division_name"`
	DivisionCode string  `json:"division_code" db:"division_code"`
	CategoryName string  `json:"category_name" db:"category_name"`
	CategoryCode string  `json:"category_code" db:"category_code"`
	QualScore    *int    `json:"qual_score" db:"qual_score"`
	QualRank     *int    `json:"qual_rank" db:"qual_rank"`
}

// RegisterParticipantRequest represents the request to register an archer to an event
type RegisterParticipantRequest struct {
	ArcherID      *string  `json:"archer_id"`
	EventArcherID *string  `json:"event_archer_id"`
	CategoryID    string   `json:"category_id" binding:"required"`
	BackNumber    *string  `json:"back_number"`
	TargetNumber  *string  `json:"target_number"`
	Session       *int     `json:"session"`
	PaymentStatus *string  `json:"payment_status" binding:"omitempty,oneof=menunggu_acc belum_lunas lunas"`
	PaymentAmount *float64 `json:"payment_amount"`
	Notes         *string  `json:"notes"`
}

// BulkImportArcher represents an archer record for bulk import
type BulkImportArcher struct {
	FullName    string `json:"full_name"`
	DateOfBirth string `json:"date_of_birth"`
	Gender      string `json:"gender"`
	City        string `json:"city"`
	Club        string `json:"club"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Division    string `json:"division"`
	Category    string `json:"category"`
}

// EventArcher represents an archer that only belongs to a single event
type EventArcher struct {
	UUID        string     `json:"id" db:"uuid"`
	EventID     string     `json:"event_id" db:"event_id"`
	FullName    string     `json:"full_name" db:"full_name"`
	Username    *string    `json:"username" db:"username"`
	Email       *string    `json:"email" db:"email"`
	Phone       *string    `json:"phone" db:"phone"`
	DateOfBirth *time.Time `json:"date_of_birth" db:"date_of_birth"`
	Gender      *string    `json:"gender" db:"gender"`
	BowType     *string    `json:"bow_type" db:"bow_type"`
	City        *string    `json:"city" db:"city"`
	School      *string    `json:"school" db:"school"`
	Club        *string    `json:"club" db:"club"`
	ClubID      *string    `json:"club_id" db:"club_id"`
	Address     *string    `json:"address" db:"address"`
	AvatarURL   *string    `json:"avatar_url" db:"avatar_url"`
	Notes       *string    `json:"notes" db:"notes"`
	Status      string     `json:"status" db:"status"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateEventArcherRequest represents payload for creating event-only archer
type CreateEventArcherRequest struct {
	FullName    string     `json:"full_name" binding:"required,min=2,max=100"`
	Username    *string    `json:"username"`
	Email       *string       `json:"email" binding:"omitempty,email"`
	Phone       *string       `json:"phone"`
	DateOfBirth *FlexibleTime `json:"date_of_birth"`
	Gender      *string       `json:"gender" binding:"omitempty,oneof=male female M F"`
	BowType     *string    `json:"bow_type" binding:"omitempty,oneof=recurve compound barebow traditional"`
	City        *string    `json:"city"`
	School      *string    `json:"school"`
	Club        *string    `json:"club"`
	ClubID      *string    `json:"club_id"`
	Address     *string    `json:"address"`
	AvatarURL   *string    `json:"avatar_url"`
	Notes       *string    `json:"notes"`
}
