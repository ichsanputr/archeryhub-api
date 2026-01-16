package models

import (
	"time"
)

// Tournament represents an archery tournament/competition
type Tournament struct {
	ID           string    `json:"id" db:"id"`
	Code         string    `json:"code" db:"code"`
	Name         string    `json:"name" db:"name"`
	ShortName    *string   `json:"short_name" db:"short_name"`
	Venue        *string   `json:"venue" db:"venue"`
	Location     *string   `json:"location" db:"location"`
	Country      *string   `json:"country" db:"country"`
	Latitude     *float64  `json:"latitude" db:"latitude"`
	Longitude    *float64  `json:"longitude" db:"longitude"`
	StartDate    time.Time `json:"start_date" db:"start_date"`
	EndDate      time.Time `json:"end_date" db:"end_date"`
	Description  *string   `json:"description" db:"description"`
	BannerURL    *string   `json:"banner_url" db:"banner_url"`
	LogoURL      *string   `json:"logo_url" db:"logo_url"`
	Type         *string   `json:"type" db:"type"` // Indoor, Outdoor, Field, 3D
	NumDistances int       `json:"num_distances" db:"num_distances"`
	NumSessions  int       `json:"num_sessions" db:"num_sessions"`
	Status       string    `json:"status" db:"status"` // draft, published, ongoing, completed, archived
	OrganizerID  *string   `json:"organizer_id" db:"organizer_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// TournamentWithDetails includes organizer information
type TournamentWithDetails struct {
	Tournament
	OrganizerName    *string `json:"organizer_name" db:"organizer_name"`
	OrganizerEmail   *string `json:"organizer_email" db:"organizer_email"`
	ParticipantCount int     `json:"participant_count" db:"participant_count"`
	EventCount       int     `json:"event_count" db:"event_count"`
}

// CreateTournamentRequest represents the request payload for creating a tournament
type CreateTournamentRequest struct {
	Code         string    `json:"code" binding:"required,min=2,max=20"`
	Name         string    `json:"name" binding:"required,min=3,max=200"`
	ShortName    *string   `json:"short_name"`
	Venue        *string   `json:"venue"`
	Location     *string   `json:"location"`
	Country      *string   `json:"country"`
	Latitude     *float64  `json:"latitude"`
	Longitude    *float64  `json:"longitude"`
	StartDate    time.Time `json:"start_date" binding:"required"`
	EndDate      time.Time `json:"end_date" binding:"required"`
	Description  *string   `json:"description"`
	BannerURL    *string   `json:"banner_url"`
	LogoURL      *string   `json:"logo_url"`
	Type         *string   `json:"type"`
	NumDistances int       `json:"num_distances"`
	NumSessions  int       `json:"num_sessions"`
}

// UpdateTournamentRequest represents the request payload for updating a tournament
type UpdateTournamentRequest struct {
	Name         *string    `json:"name"`
	ShortName    *string    `json:"short_name"`
	Venue        *string    `json:"venue"`
	Location     *string    `json:"location"`
	Country      *string    `json:"country"`
	Latitude     *float64   `json:"latitude"`
	Longitude    *float64   `json:"longitude"`
	StartDate    *time.Time `json:"start_date"`
	EndDate      *time.Time `json:"end_date"`
	Description  *string    `json:"description"`
	BannerURL    *string    `json:"banner_url"`
	LogoURL      *string    `json:"logo_url"`
	Type         *string    `json:"type"`
	NumDistances *int       `json:"num_distances"`
	NumSessions  *int       `json:"num_sessions"`
	Status       *string    `json:"status"`
}

// TournamentEvent represents an event within a tournament (division + category)
type TournamentEvent struct {
	ID                  string    `json:"id" db:"id"`
	TournamentID        string    `json:"tournament_id" db:"tournament_id"`
	DivisionID          string    `json:"division_id" db:"division_id"`
	CategoryID          string    `json:"category_id" db:"category_id"`
	MaxParticipants     int       `json:"max_participants" db:"max_participants"`
	QualificationArrows int       `json:"qualification_arrows" db:"qualification_arrows"`
	EliminationFormat   string    `json:"elimination_format" db:"elimination_format"`
	TeamEvent           bool      `json:"team_event" db:"team_event"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
}

// TournamentEventWithDetails includes division and category details
type TournamentEventWithDetails struct {
	TournamentEvent
	DivisionName     string `json:"division_name" db:"division_name"`
	DivisionCode     string `json:"division_code" db:"division_code"`
	CategoryName     string `json:"category_name" db:"category_name"`
	CategoryCode     string `json:"category_code" db:"category_code"`
	ParticipantCount int    `json:"participant_count" db:"participant_count"`
}

// Division represents a bow division (Recurve, Compound, Barebow, etc.)
type Division struct {
	ID           string    `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Code         string    `json:"code" db:"code"`
	Description  *string   `json:"description" db:"description"`
	DisplayOrder int       `json:"display_order" db:"display_order"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Category represents an age/gender category
type Category struct {
	ID           string    `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Code         string    `json:"code" db:"code"`
	AgeFrom      *int      `json:"age_from" db:"age_from"`
	AgeTo        *int      `json:"age_to" db:"age_to"`
	Gender       string    `json:"gender" db:"gender"` // M, F, X
	DisplayOrder int       `json:"display_order" db:"display_order"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Session represents a competition session
type Session struct {
	ID                string     `json:"id" db:"id"`
	TournamentID      string     `json:"tournament_id" db:"tournament_id"`
	SessionOrder      int        `json:"session_order" db:"session_order"`
	Name              *string    `json:"name" db:"name"`
	SessionDate       *time.Time `json:"session_date" db:"session_date"`
	StartTime         *string    `json:"start_time" db:"start_time"`
	EndTime           *string    `json:"end_time" db:"end_time"`
	NumTargets        int        `json:"num_targets" db:"num_targets"`
	AthletesPerTarget int        `json:"athletes_per_target" db:"athletes_per_target"`
	Locked            bool       `json:"locked" db:"locked"`
	Notes             *string    `json:"notes" db:"notes"`
}

// Distance represents shooting distance configuration
type Distance struct {
	ID            string  `json:"id" db:"id"`
	TournamentID  string  `json:"tournament_id" db:"tournament_id"`
	EventID       string  `json:"event_id" db:"event_id"`
	DistanceOrder int     `json:"distance_order" db:"distance_order"`
	DistanceValue int     `json:"distance_value" db:"distance_value"` // in meters
	ArrowsPerEnd  int     `json:"arrows_per_end" db:"arrows_per_end"`
	NumEnds       int     `json:"num_ends" db:"num_ends"`
	TargetFace    *string `json:"target_face" db:"target_face"` // 122cm, 80cm, etc.
}
