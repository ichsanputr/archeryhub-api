package models

import (
	"time"
)

// Device represents a scoring device (tablet/phone)
type Device struct {
	ID               string     `json:"id" db:"id"`
	TournamentID     string     `json:"tournament_id" db:"tournament_id"`
	DeviceCode       string     `json:"device_code" db:"device_code"`
	DeviceName       *string    `json:"device_name" db:"device_name"`
	DeviceType       string     `json:"device_type" db:"device_type"` // tablet, phone, scorekeeper, display
	PIN              *string    `json:"pin" db:"pin"`
	QRPayload        *string    `json:"qr_payload" db:"qr_payload"`
	TargetAssignment *string    `json:"target_assignment" db:"target_assignment"`
	Session          *int       `json:"session" db:"session"`
	LastSync         *time.Time `json:"last_sync" db:"last_sync"`
	Status           string     `json:"status" db:"status"` // active, inactive, blocked
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// RegisterDeviceRequest represents the request to register a device
type RegisterDeviceRequest struct {
	DeviceCode       string  `json:"device_code" binding:"required,min=3,max=20"`
	DeviceName       *string `json:"device_name"`
	DeviceType       string  `json:"device_type" binding:"required,oneof=tablet phone scorekeeper display"`
	TargetAssignment *string `json:"target_assignment"`
	Session          *int    `json:"session"`
}

// DeviceConfigResponse represents the configuration sent to a device
type DeviceConfigResponse struct {
	Device
	TournamentName string                       `json:"tournament_name"`
	TournamentCode string                       `json:"tournament_code"`
	Events         []TournamentEventWithDetails `json:"events"`
	Distances      []Distance                   `json:"distances"`
	Participants   []ParticipantWithDetails     `json:"participants"`
}

// DeviceSyncRequest represents data synced from a device
type DeviceSyncRequest struct {
	DeviceID string               `json:"device_id" binding:"required"`
	Scores   []SubmitScoreRequest `json:"scores"`
	LastSync time.Time            `json:"last_sync"`
}

// QRCodePayload represents the data encoded in QR code for device setup
type QRCodePayload struct {
	TournamentID string `json:"tournament_id"`
	DeviceCode   string `json:"device_code"`
	PIN          string `json:"pin"`
	APIEndpoint  string `json:"api_endpoint"`
}
