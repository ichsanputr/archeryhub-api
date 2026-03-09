package models

import "time"

type Club struct {
	UUID        string    `json:"uuid" db:"uuid"`
	Name        string    `json:"name" db:"name"`
	Slug        string    `json:"slug" db:"slug"`
	LogoURL     *string   `json:"logo_url" db:"logo_url"`
	BannerURL   *string   `json:"banner_url" db:"banner_url"`
	City        *string   `json:"city" db:"city"`
	Address     *string   `json:"address" db:"address"`
	Description *string   `json:"description" db:"description"`
	Status      string    `json:"status" db:"status"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
