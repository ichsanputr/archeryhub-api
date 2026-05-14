package models

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// FlexibleTime handles empty strings and RFC3339 dates during JSON unmarshaling
type FlexibleTime struct {
	time.Time
}

// Value implements the driver.Valuer interface
func (ft FlexibleTime) Value() (driver.Value, error) {
	if ft.IsZero() {
		return nil, nil
	}
	return ft.Time, nil
}

// Scan implements the sql.Scanner interface
func (ft *FlexibleTime) Scan(value interface{}) error {
	if value == nil {
		ft.Time = time.Time{}
		return nil
	}
	t, ok := value.(time.Time)
	if !ok {
		return fmt.Errorf("failed to scan FlexibleTime: expected time.Time, got %T", value)
	}
	ft.Time = t
	return nil
}

func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")
	if s == "" || s == "null" {
		ft.Time = time.Time{}
		return nil
	}

	// Try RFC3339 first
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		ft.Time = t
		return nil
	}

	// Try standard date format
	t, err = time.Parse("2006-01-02", s)
	if err == nil {
		ft.Time = t
		return nil
	}

	return err
}

// Event represents an archery Event/competition
type Event struct {
	UUID                  string     `json:"id" db:"uuid" example:"evt-8f3c2a14-2b73-4a7f-8f7f-2ef1e6c1159a"`
	Code                  string     `json:"code" db:"code" example:"AHC-2026-JKT-OPEN"`
	Name                  string     `json:"name" db:"name" example:"ArcheryHub Jakarta Open 2026"`
	ShortName             *string    `json:"short_name" db:"short_name" example:"JKT Open 2026"`
	Slug                  string     `json:"slug" db:"slug" example:"archeryhub-jakarta-open-2026"`
	Venue                 *string    `json:"venue" db:"venue" example:"Lapangan ABC Senayan"`
	Address               *string    `json:"address" db:"address" example:"Jl. Asia Afrika No.8, Jakarta"`
	GmapLink              *string    `json:"gmaps_link" db:"gmaps_link" example:"https://maps.google.com/?q=-6.2185,106.8022"`
	Location              *string    `json:"location" db:"location" example:"Jakarta Selatan"`
	City                  *string    `json:"city" db:"city" example:"Jakarta"`
	StartDate             *time.Time `json:"start_date" db:"start_date" swaggertype:"string" format:"date-time" example:"2026-05-18T08:00:00Z"`
	EndDate               *time.Time `json:"end_date" db:"end_date" swaggertype:"string" format:"date-time" example:"2026-05-21T17:00:00Z"`
	RegistrationDeadline  *time.Time `json:"registration_deadline" db:"registration_deadline" swaggertype:"string" format:"date-time" example:"2026-05-10T23:59:59Z"`
	Description           *string    `json:"description" db:"description" example:"Kejuaraan terbuka nasional untuk kategori recurve, compound, dan barebow."`
	BannerURL             *string    `json:"banner_url" db:"banner_url" example:"https://cdn.archeryhub.id/media/banner-jkt-open-2026.jpg"`
	LogoURL               *string    `json:"logo_url" db:"logo_url" example:"https://cdn.archeryhub.id/media/logo-jkt-open-2026.png"`
	Type                  *string    `json:"type" db:"type" example:"Outdoor"`                   // Indoor, Outdoor, Field, 3D (kept for backward compatibility)
	LocationType          *string    `json:"location_type" db:"location_type" example:"Outdoor"` // Location type: Indoor, Outdoor, Field, 3D, etc.
	NumDistances          *int       `json:"num_distances" db:"num_distances" example:"2"`
	NumSessions           *int       `json:"num_sessions" db:"num_sessions" example:"4"`
	EntryFee              float64    `json:"entry_fee" db:"entry_fee" example:"350000"`
	Status                string     `json:"status" db:"status" example:"published"` // draft, active, published
	OrganizerID           *string    `json:"organizer_id" db:"organizer_id" example:"org-1fa53fca-b7da-4be6-b6eb-8b962d2f7d55"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at" swaggertype:"string" format:"date-time" example:"2026-03-01T10:30:00Z"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at" swaggertype:"string" format:"date-time" example:"2026-03-12T13:10:00Z"`
	TotalPrize            float64    `json:"total_prize" db:"total_prize" example:"50000000"`
	TechnicalGuidebookURL *string    `json:"technical_guidebook_url" db:"technical_guidebook_url" example:"https://cdn.archeryhub.id/media/technical-guidebook-jkt-open-2026.pdf"`
	PageSettings          *string    `json:"page_settings_raw" db:"page_settings"`
	FAQ                   *string    `json:"faq_raw" db:"faq"`
}

// EventWithDetails includes organizer information
type EventWithDetails struct {
	Event
	OrganizerName         *string                    `json:"organizer_name" db:"organizer_name" example:"ArcheryHub Club Jakarta"`
	OrganizerEmail        *string                    `json:"organizer_email" db:"organizer_email" example:"event@archeryhub.id"`
	OrganizerAvatarURL    *string                    `json:"organizer_avatar_url" db:"organizer_avatar_url" example:"https://cdn.archeryhub.id/media/organizer-avatar.png"`
	OrganizerSlug         *string                    `json:"organizer_slug" db:"organizer_slug" example:"archeryhub-jakarta"`
	OrganizerPhone        *string                    `json:"organizer_phone" db:"organizer_phone" example:"081234567890"`
	ParticipantCount      int                        `json:"participant_count" db:"participant_count" example:"128"`
	EventCount            int                        `json:"event_count" db:"event_count" example:"12"`
	AccreditationStatus   *string                    `json:"accreditation_status" db:"accreditation_status" example:"approved"`
	PaymentStatus         *string                    `json:"payment_status" db:"payment_status" example:"paid"`
	ParticipantUUID       *string                    `json:"participant_uuid" db:"participant_uuid" example:"par-6f0bf699-d807-4ad4-a50d-5d60f7f7ad5d"`
	ParticipantStatus     *string                    `json:"participant_status" db:"participant_status" example:"registered"`
	QRRaw                 *string                    `json:"qr_raw" db:"qr_raw" example:"EVT2026-ARC-0001"`
	WhatsAppNumber        *string                    `json:"whatsapp_number" db:"whatsapp_number" example:"081234567890"`
	VenueType             *string                    `json:"venue_type" db:"venue_type" example:"outdoor"`
	TargetCount           int                        `json:"target_count" db:"target_count" example:"48"`
	ActiveTargetCount     int                        `json:"active_target_count" db:"active_target_count" example:"36"`
	PageSettings          any                        `json:"page_settings,omitempty" db:"-"`
	FAQ                   any                        `json:"faq,omitempty" db:"-"`
	LocationDetail        EventLocationDetail        `json:"location_detail,omitempty"`
	Participants          []EventParticipantPreview  `json:"participants,omitempty"`
	Schedules             []EventSchedule            `json:"schedules,omitempty"`
	Results               []EventResultPreview       `json:"results,omitempty"`
	Gallery               []EventImage               `json:"gallery,omitempty"`
	CompetitionCategories []EventCompetitionCategory `json:"competition_categories,omitempty"`
}

type EventLocationDetail struct {
	Venue        *string `json:"venue" example:"Lapangan ABC Senayan"`
	Address      *string `json:"address" example:"Jl. Asia Afrika No.8, Jakarta"`
	GmapLink     *string `json:"gmaps_link" example:"https://maps.google.com/?q=-6.2185,106.8022"`
	Location     *string `json:"location" example:"Jakarta Selatan"`
	City         *string `json:"city" example:"Jakarta"`
	LocationType *string `json:"location_type" example:"Outdoor"`
}

type EventParticipantPreview struct {
	ParticipantID string  `json:"participant_id" db:"participant_id" example:"par-6f0bf699-d807-4ad4-a50d-5d60f7f7ad5d"`
	ArcherID      *string `json:"archer_id" db:"archer_id" example:"arc-a49ee7d7-9d7b-4be7-8652-342f2fca23f9"`
	FullName      string  `json:"full_name" db:"full_name" example:"Rizky Pratama"`
	ClubName      *string `json:"club_name" db:"club_name" example:"ArcheryHub Club Jakarta"`
	CategoryName  *string `json:"category_name" db:"category_name" example:"Recurve Umum Putra"`
	PaymentStatus string  `json:"payment_status" db:"payment_status" example:"lunas"`
	QualRank      *int    `json:"qual_rank" db:"qual_rank" example:"1"`
	QualScore     *int    `json:"qual_score" db:"qual_score" example:"668"`
	AvatarURL     *string `json:"avatar_url" db:"avatar_url" example:"https://cdn.archeryhub.id/media/archer/rizky-pratama.jpg"`
}

type EventResultPreview struct {
	ParticipantID string  `json:"participant_id" db:"participant_id" example:"par-6f0bf699-d807-4ad4-a50d-5d60f7f7ad5d"`
	FullName      string  `json:"full_name" db:"full_name" example:"Rizky Pratama"`
	CategoryName  *string `json:"category_name" db:"category_name" example:"Recurve Umum Putra"`
	Rank          *int    `json:"rank" db:"rank" example:"1"`
	Score         *int    `json:"score" db:"score" example:"668"`
	XCount        int     `json:"x_count" db:"x_count" example:"22"`
}

type EventCompetitionCategory struct {
	CategoryID         string  `json:"category_id" db:"category_id" example:"cat-44f67d53-032f-428f-a8f2-8db5672a7a9d"`
	DivisionName       *string `json:"division_name" db:"division_name" example:"Recurve"`
	CategoryName       *string `json:"category_name" db:"category_name" example:"Umum"`
	EventTypeName      *string `json:"event_type_name" db:"event_type_name" example:"Individual"`
	GenderDivisionName *string `json:"gender_division_name" db:"gender_division_name" example:"Putra"`
	ParticipantCount   int     `json:"participant_count" db:"participant_count" example:"64"`
}

// CreateEventRequest represents the request payload for creating a Event
type CreateEventRequest struct {
	Code                  string                    `json:"code" binding:"omitempty,max=20"`
	Name                  string                    `json:"name"`
	Slug                  string                    `json:"slug"`
	ShortName             *string                   `json:"short_name"`
	Venue                 *string                   `json:"venue"`
	GmapLink              *string                   `json:"gmaps_link"`
	Location              *string                   `json:"location"`
	City                  *string                   `json:"city"`
	StartDate             FlexibleTime              `json:"start_date"`
	EndDate               FlexibleTime              `json:"end_date"`
	Description           *string                   `json:"description"`
	BannerURL             *string                   `json:"banner_url"`
	LogoURL               *string                   `json:"logo_url"`
	Type                  *string                   `json:"type"` // Deprecated, use location_type
	LocationType          *string                   `json:"location_type"`
	NumDistances          *int                      `json:"num_distances"`
	NumSessions           *int                      `json:"num_sessions"`
	EntryFee              float64                   `json:"entry_fee"`
	RegistrationDeadline  FlexibleTime              `json:"registration_deadline"`
	Status                string                    `json:"status"`
	Divisions             []string                  `json:"divisions"`
	Categories            []string                  `json:"categories"`
	Images                []CreateEventImageRequest `json:"images"`
	TotalPrize            float64                   `json:"total_prize"`
	TechnicalGuidebookURL *string                   `json:"technical_guidebook_url"`
	PageSettings          *string                   `json:"page_settings"`
	FAQ                   interface{}               `json:"faq"`
}

// UpdateEventRequest represents the request payload for updating a Event
type UpdateEventRequest struct {
	Name                  *string       `json:"name"`
	ShortName             *string       `json:"short_name"`
	Venue                 *string       `json:"venue"`
	Address               *string       `json:"address"`
	GmapLink              *string       `json:"gmaps_link"`
	Location              *string       `json:"location"`
	City                  *string       `json:"city"`
	StartDate             *FlexibleTime `json:"start_date"`
	EndDate               *FlexibleTime `json:"end_date"`
	Description           *string       `json:"description"`
	BannerURL             *string       `json:"banner_url"`
	LogoURL               *string       `json:"logo_url"`
	Type                  *string       `json:"type"` // Deprecated, use location_type
	LocationType          *string       `json:"location_type"`
	NumDistances          *int          `json:"num_distances"`
	NumSessions           *int          `json:"num_sessions"`
	Status                *string       `json:"status"`
	EntryFee              *float64      `json:"entry_fee"`
	RegistrationDeadline  *FlexibleTime `json:"registration_deadline"`
	TotalPrize            *float64      `json:"total_prize"`
	TechnicalGuidebookURL *string       `json:"technical_guidebook_url"`
	PageSettings          *string       `json:"page_settings"`
	FAQ                   interface{}   `json:"faq"`
}

// EventEvent represents an event within a Event (division + category)
type EventEvent struct {
	UUID                string    `json:"id" db:"uuid"`
	EventID             string    `json:"event_id" db:"event_id"`
	DivisionUUID        string    `json:"division_id" db:"division_uuid"`
	CategoryUUID        string    `json:"category_id" db:"category_uuid"`
	MaxParticipants     int       `json:"max_participants" db:"max_participants"`
	TeamSize            int       `json:"team_size" db:"team_size"`
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
	UUID             string     `json:"id" db:"uuid"`
	EventID          string     `json:"Event_id" db:"Event_id"`
	SessionOrder     int        `json:"session_order" db:"session_order"`
	Name             *string    `json:"name" db:"name"`
	SessionDate      *time.Time `json:"session_date" db:"session_date"`
	StartTime        *string    `json:"start_time" db:"start_time"`
	EndTime          *string    `json:"end_time" db:"end_time"`
	NumTargets       int        `json:"num_targets" db:"num_targets"`
	ArchersPerTarget int        `json:"archers_per_target" db:"archers_per_target"`
	Locked           bool       `json:"locked" db:"locked"`
	Notes            *string    `json:"notes" db:"notes"`
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

// EventTarget represents a physical target in an event
type EventTarget struct {
	UUID        string    `json:"id" db:"uuid"`
	EventUUID   string    `json:"event_id" db:"event_uuid"`
	TargetName  string    `json:"target_name" db:"target_name"`
	BoardNumber int       `json:"board_number" db:"board_number"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// QualificationSessionScore represents scores for a specific session in qualification
type QualificationSessionScore struct {
	AssignmentUUID string `json:"assignment_id"`
	SessionCode    string `json:"session_code"`
	SessionName    string `json:"session_name"`
	EndScores      string `json:"end_scores"`
	TotalEnds      int    `json:"total_ends"`
	TotalScore     int    `json:"total_score"`
	TotalTenX      int    `json:"total_10x"`
	TotalX         int    `json:"total_x"`
}

// QualificationEntry represents a single participant's result in qualification leaderboard
type QualificationEntry struct {
	Rank            int                         `json:"rank"`
	ParticipantUUID string                      `json:"participant_id"`
	ArcherUUID      string                      `json:"archer_uuid"`
	ArcherName      string                      `json:"archer_name"`
	AvatarURL       *string                     `json:"avatar_url"`
	ClubName        *string                     `json:"club_name"`
	TotalScore      int                         `json:"total_score"`
	TotalTenX       int                         `json:"total_10x"`
	TotalX          int                         `json:"total_x"`
	EndsCompleted   int                         `json:"ends_completed"`
	Sessions        []QualificationSessionScore `json:"sessions"`
}

// QualificationResultsResponse represents the full leaderboard for qualification
type QualificationResultsResponse struct {
	TotalCumulativeEnds int                  `json:"total_cumulative_ends"`
	Leaderboard         []QualificationEntry `json:"leaderboard"`
}
