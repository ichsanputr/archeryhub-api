package models

import "time"

type Scorekeeper struct {
	UUID             string    `json:"uuid" db:"uuid"`
	OrganizationUUID string    `json:"organization_uuid" db:"organization_uuid"`
	Name             string    `json:"name" db:"name"`
	Email            string    `json:"email" db:"email"`
	Password         string    `json:"password,omitempty" db:"password"`
	GoogleID         *string   `json:"google_id,omitempty" db:"google_id"`
	AvatarURL        *string   `json:"avatar_url" db:"avatar_url"`
	Status           string    `json:"status" db:"status"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}
