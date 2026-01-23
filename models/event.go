package models

import (
	"strings"
	"time"
)

// FlexibleTime handles empty strings and RFC3339 dates during JSON unmarshaling
type FlexibleTime struct {
	time.Time
}

func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")
	if s == "" || s == "null" {
		ft.Time = time.Time{}
		return nil
	}
	
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Try other common formats if needed, or just return err
		return err
	}
	ft.Time = t
	return nil
}

// Event represents an archery Event/competition
type Event struct {
	UUID         string    `json:"id" db:"uuid"` 
	Code         string    `json:"code" db:"code"`
	Name         string    `json:"name" db:"name"`
	ShortName    *string   `json:"short_name" db:"short_name"`
	Slug         string    `json:"slug" db:"slug"`
	Venue        *string   `json:"venue" db:"venue"`
	Address      *string   `json:"address" db:"address"`
	GmapLink     *string   `json:"gmaps_link" db:"gmaps_link"`
	Location     *string   `json:"location" db:"location"`
	Country      *string   `json:"country" db:"country"`
	Latitude     *float64  `json:"latitude" db:"latitude"`
	Longitude    *float64  `json:"longitude" db:"longitude"`
	StartDate            *time.Time `json:"start_date" db:"start_date"`
	EndDate              *time.Time `json:"end_date" db:"end_date"`
	RegistrationDeadline *time.Time `json:"registration_deadline" db:"registration_deadline"`
	Description          *string    `json:"description" db:"description"`
	BannerURL            *string    `json:"banner_url" db:"banner_url"`
	LogoURL              *string    `json:"logo_url" db:"logo_url"`
	Type                 *string    `json:"type" db:"type"` // Indoor, Outdoor, Field, 3D
	NumDistances         *int       `json:"num_distances" db:"num_distances"`
	NumSessions          *int       `json:"num_sessions" db:"num_sessions"`
	EntryFee             float64    `json:"entry_fee" db:"entry_fee"`
	MaxParticipants      *int       `json:"max_participants" db:"max_participants"`
	Status               string     `json:"status" db:"status"` // draft, published, ongoing, completed, archived
	OrganizerID          *string    `json:"organizer_id" db:"organizer_id"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
	DisciplineName       string     `json:"discipline_name" db:"discipline_name"`
}

// EventWithDetails includes organizer information
type EventWithDetails struct {
	Event
	OrganizerName      *string `json:"organizer_name" db:"organizer_name"`
	OrganizerEmail     *string `json:"organizer_email" db:"organizer_email"`
	ParticipantCount   int     `json:"participant_count" db:"participant_count"`
	EventCount         int     `json:"event_count" db:"event_count"`
	AccreditationStatus *string `json:"accreditation_status" db:"accreditation_status"`
	PaymentStatus      *string `json:"payment_status" db:"payment_status"`
	ParticipantUUID    *string `json:"participant_uuid" db:"participant_uuid"`
}

// CreateEventRequest represents the request payload for creating a Event
type CreateEventRequest struct {
	Code                 string       `json:"code" binding:"required,min=2,max=20"`
	Name                 string       `json:"name"`
	ShortName            *string      `json:"short_name"`
	Venue                *string      `json:"venue"`
	GmapLink             *string      `json:"gmaps_link"`
	Location             *string      `json:"location"`
	Country              *string      `json:"country"`
	Latitude             *float64     `json:"latitude"`
	Longitude            *float64     `json:"longitude"`
	StartDate            FlexibleTime `json:"start_date"`
	EndDate              FlexibleTime `json:"end_date"`
	Description          *string      `json:"description"`
	BannerURL            *string      `json:"banner_url"`
	LogoURL              *string      `json:"logo_url"`
	Type                 *string      `json:"type"`
	NumDistances         *int         `json:"num_distances"`
	NumSessions          *int         `json:"num_sessions"`
	EntryFee             float64      `json:"entry_fee"`
	MaxParticipants      *int         `json:"max_participants"`
	RegistrationDeadline FlexibleTime `json:"registration_deadline"`
	Status               string       `json:"status"`
	Divisions            []string     `json:"divisions"`
	Categories           []string     `json:"categories"`
	Images               []CreateEventImageRequest `json:"images"`
}

// UpdateEventRequest represents the request payload for updating a Event
type UpdateEventRequest struct {
	Name         *string    `json:"name"`
	ShortName    *string    `json:"short_name"`
	Venue        *string    `json:"venue"`
	Address      *string    `json:"address"`
	GmapLink     *string    `json:"gmaps_link"`
	Location     *string    `json:"location"`
	Country      *string    `json:"country"`
	Latitude     *float64   `json:"latitude"`
	Longitude    *float64   `json:"longitude"`
	StartDate    *FlexibleTime `json:"start_date"`
	EndDate      *FlexibleTime `json:"end_date"`
	Description  *string    `json:"description"`
	BannerURL    *string    `json:"banner_url"`
	LogoURL      *string    `json:"logo_url"`
	Type         *string    `json:"type"`
	NumDistances *int       `json:"num_distances"`
	NumSessions  *int       `json:"num_sessions"`
	Status       *string    `json:"status"`
	EntryFee     *float64   `json:"entry_fee"`
	MaxParticipants *int    `json:"max_participants"`
	RegistrationDeadline *FlexibleTime `json:"registration_deadline"`
}

// EventEvent represents an event within a Event (division + category)
type EventEvent struct {
	UUID                string    `json:"id" db:"uuid"`
	EventID        string    `json:"event_id" db:"event_id"`
	DivisionUUID          string    `json:"division_id" db:"division_uuid"`
	CategoryUUID          string    `json:"category_id" db:"category_uuid"`
	MaxParticipants     int       `json:"max_participants" db:"max_participants"`
	QualificationArrows int       `json:"qualification_arrows" db:"qualification_arrows"`
	EliminationFormat   string    `json:"elimination_format" db:"elimination_format"`
	TeamEvent           bool      `json:"team_event" db:"team_event"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
}

// EventEventWithDetails includes division and category details
type EventEventWithDetails struct {
	EventEvent
	DivisionName     string `json:"division_name" db:"division_name"`
	CategoryName     string `json:"category_name" db:"category_name"`
	ParticipantCount int    `json:"participant_count" db:"participant_count"`
}


// Session represents a competition session
type Session struct {
	UUID              string     `json:"id" db:"uuid"`
	EventID      string     `json:"Event_id" db:"Event_id"`
	SessionOrder      int        `json:"session_order" db:"session_order"`
	Name              *string    `json:"name" db:"name"`
	SessionDate       *time.Time `json:"session_date" db:"session_date"`
	StartTime         *string    `json:"start_time" db:"start_time"`
	EndTime           *string    `json:"end_time" db:"end_time"`
	NumTargets        int        `json:"num_targets" db:"num_targets"`
	ArchersPerTarget  int        `json:"archers_per_target" db:"archers_per_target"`
	Locked            bool       `json:"locked" db:"locked"`
	Notes             *string    `json:"notes" db:"notes"`
}

// EventSchedule represents a schedule item for an event
type EventSchedule struct {
	UUID        string     `json:"id" db:"uuid"`
	EventID     string     `json:"event_id" db:"event_id"`
	Title       string     `json:"title" db:"title"`
	Description *string    `json:"description" db:"description"`
	StartTime   time.Time  `json:"start_time" db:"start_time"`
	EndTime     *time.Time `json:"end_time" db:"end_time"`
	DayOrder    *int       `json:"day_order" db:"day_order"`
	SortOrder   *int       `json:"sort_order" db:"sort_order"`
	Location    *string    `json:"location" db:"location"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// EventCategoryRef represents a reusable event category (bow type + age group)
type EventCategoryRef struct {
	UUID       string `json:"id" db:"uuid"`
	Name       string `json:"name" db:"name"`
	BowTypeID  string `json:"bow_type_id" db:"bow_type_id"`
	BowName    string `json:"bow_name" db:"bow_name"`
	AgeGroupID string `json:"age_group_id" db:"age_group_id"`
	AgeName    string `json:"age_name" db:"age_name"`
	Status     string `json:"status" db:"status"`
}


