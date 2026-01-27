package models

import "time"

// Club represents an archery club
type Club struct {
	UUID               string     `json:"uuid" db:"uuid"`
	Slug               string     `json:"slug" db:"slug"`
	Name               string     `json:"name" db:"name"`
	Abbreviation       *string    `json:"abbreviation" db:"abbreviation"`
	Description        *string    `json:"description" db:"description"`
	BannerURL          *string    `json:"banner_url" db:"banner_url"`
	AvatarURL          *string    `json:"avatar_url" db:"avatar_url"`
	Email              string     `json:"email" db:"email"`
	Phone              *string    `json:"phone" db:"phone"`
	Address            *string    `json:"address" db:"address"`
	City               *string    `json:"city" db:"city"`
	Province           *string    `json:"province" db:"province"`
	EstablishedYear    *int       `json:"established_year" db:"established_year"`
	HeadCoachName      *string    `json:"head_coach_name" db:"head_coach_name"`
	HeadCoachPhone     *string    `json:"head_coach_phone" db:"head_coach_phone"`
	TrainingSchedule   *string    `json:"training_schedule" db:"training_schedule"`
	Website            *string    `json:"website" db:"website"`
	SocialFacebook     *string    `json:"social_facebook" db:"social_facebook"`
	SocialInstagram    *string    `json:"social_instagram" db:"social_instagram"`
	VerificationStatus string     `json:"verification_status" db:"verification_status"`
	Status             string     `json:"status" db:"status"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

// UpdateClubRequest represents the request to update club profile
type UpdateClubRequest struct {
	Name             *string `json:"clubName"`
	Abbreviation     *string `json:"abbreviation"`
	Description      *string `json:"description"`
	BannerURL        *string `json:"banner_url"`
	AvatarURL        *string `json:"avatar_url"`
	Address          *string `json:"address"`
	City             *string `json:"city"`
	Province         *string `json:"province"`
	Phone            *string `json:"phone"`
	Email            *string `json:"email"`
	Website          *string `json:"website"`
	SocialFacebook   *string `json:"social_facebook"`
	SocialInstagram  *string `json:"social_instagram"`
	TrainingSchedule *string `json:"training_schedule"`
	HeadCoachName    *string `json:"head_coach_name"`
	HeadCoachPhone   *string `json:"head_coach_phone"`
	EstablishedDate  *string `json:"establishedDate"` // Mapping from frontend date input
}
