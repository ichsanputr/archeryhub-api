package models

import "time"

type Club struct {
	UUID         string    `json:"uuid" db:"uuid"`
	Slug         string    `json:"slug" db:"slug"`
	Name         string    `json:"name" db:"name"`
	Abbreviation *string   `json:"abbreviation" db:"abbreviation"`
	LogoURL      *string   `json:"logo_url" db:"logo_url"`
	City         *string   `json:"city" db:"city"`
	Status       string    `json:"status" db:"status"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
