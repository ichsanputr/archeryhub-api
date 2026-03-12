package models

import "time"

type Club struct {
	UUID               string    `json:"uuid" db:"uuid"`
	Slug               string    `json:"slug" db:"slug"`
	Name               string    `json:"name" db:"name"`
	Abbreviation       *string   `json:"abbreviation" db:"abbreviation"`
	Description        *string   `json:"description" db:"description"`
	BannerURL          *string   `json:"banner_url" db:"banner_url"`
	LogoURL            *string   `json:"logo_url" db:"logo_url"`
	Phone              *string   `json:"phone" db:"phone"`
	Address            *string   `json:"address" db:"address"`
	City               *string   `json:"city" db:"city"`
	Province           *string   `json:"province" db:"province"`
	PostalCode         *string   `json:"postal_code" db:"postal_code"`
	EstablishedDate    *string   `json:"established_date" db:"established_date"`
	RegistrationNumber *string   `json:"registration_number" db:"registration_number"`
	OrganizationID     *string   `json:"organization_id" db:"organization_id"`
	HeadCoachName      *string   `json:"head_coach_name" db:"head_coach_name"`
	HeadCoachPhone     *string   `json:"head_coach_phone" db:"head_coach_phone"`
	TrainingSchedule   *string   `json:"training_schedule" db:"training_schedule"`
	Facilities         *string   `json:"facilities" db:"facilities"`
	Website            *string   `json:"website" db:"website"`
	Status             string    `json:"status" db:"status"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}
