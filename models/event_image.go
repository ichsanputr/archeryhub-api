package models

import "time"

// EventImage represents an image associated with an event
type EventImage struct {
	UUID         string    `json:"id" db:"uuid"`
	EventID      string    `json:"event_id" db:"event_id"`
	URL          string    `json:"url" db:"url"`
	Caption      *string   `json:"caption" db:"caption"`
	AltText      *string   `json:"alt_text" db:"alt_text"`
	DisplayOrder int       `json:"display_order" db:"display_order"`
	IsPrimary    bool      `json:"is_primary" db:"is_primary"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// CreateEventImageRequest represents the request payload for adding event images
type CreateEventImageRequest struct {
	URL          string  `json:"url" binding:"required"`
	Caption      *string `json:"caption"`
	AltText      *string `json:"alt_text"`
	DisplayOrder int     `json:"display_order"`
	IsPrimary    bool    `json:"is_primary"`
}

// UpdateEventImageRequest represents the request payload for updating an event image
type UpdateEventImageRequest struct {
	Caption      *string `json:"caption"`
	AltText      *string `json:"alt_text"`
	DisplayOrder *int    `json:"display_order"`
	IsPrimary    *bool   `json:"is_primary"`
}
