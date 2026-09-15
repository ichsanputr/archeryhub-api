package models

import "time"

// TournamentExternal represents an external tournament scraped from Ianseo or other sources
type TournamentExternal struct {
	ID                int64     `json:"id" db:"id"`
	UUID              string    `json:"uuid" db:"uuid"`
	Slug              string    `json:"slug" db:"slug"`
	SourcePlatform    string    `json:"source_platform" db:"source_platform"`
	ExternalID        string    `json:"external_id" db:"external_id"`
	SourceURL         string    `json:"source_url" db:"source_url"`
	Name              string    `json:"name" db:"name"`
	ShortName         *string   `json:"short_name" db:"short_name"`
	Venue             *string   `json:"venue" db:"venue"`
	Location          *string   `json:"location" db:"location"`
	City              *string   `json:"city" db:"city"`
	Country           *string   `json:"country" db:"country"`
	StartDate         *string   `json:"start_date" db:"start_date"`
	EndDate           *string   `json:"end_date" db:"end_date"`
	BannerURL         *string   `json:"banner_url" db:"banner_url"`
	LogoURL           *string   `json:"logo_url" db:"logo_url"`
	Status            string    `json:"status" db:"status"`
	CategoriesCount   int       `json:"categories_count" db:"categories_count"`
	ParticipantsCount int       `json:"participants_count" db:"participants_count"`
	DataJSON          string    `json:"data_json" db:"data_json"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// TournamentExternalListItem for listing view
type TournamentExternalListItem struct {
	ID                int64   `json:"id" db:"id"`
	UUID              string  `json:"uuid" db:"uuid"`
	Slug              string  `json:"slug" db:"slug"`
	SourcePlatform    string  `json:"source_platform" db:"source_platform"`
	ExternalID        string  `json:"external_id" db:"external_id"`
	Name              string  `json:"name" db:"name"`
	ShortName         *string `json:"short_name" db:"short_name"`
	Venue             *string `json:"venue" db:"venue"`
	Location          *string `json:"location" db:"location"`
	City              *string `json:"city" db:"city"`
	Country           *string `json:"country" db:"country"`
	StartDate         *string `json:"start_date" db:"start_date"`
	EndDate           *string `json:"end_date" db:"end_date"`
	BannerURL         *string `json:"banner_url" db:"banner_url"`
	LogoURL           *string `json:"logo_url" db:"logo_url"`
	Status            string  `json:"status" db:"status"`
	CategoriesCount   int     `json:"categories_count" db:"categories_count"`
	ParticipantsCount int     `json:"participants_count" db:"participants_count"`
}
