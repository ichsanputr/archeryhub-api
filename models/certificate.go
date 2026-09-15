package models

import (
	"time"
)

// EventCertificate represents the certificate template configuration for an event
type EventCertificate struct {
	UUID          string    `json:"id" db:"uuid"`
	EventID       string    `json:"event_id" db:"tournament_id"`
	HTMLTemplate  *string   `json:"html_template" db:"html_template"`
	BackgroundURL *string   `json:"background_url" db:"background_url"`
	SignatureURL  *string   `json:"signature_url" db:"signature_url"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// ArcherCertificate represents an issued certificate for a participant
type ArcherCertificate struct {
	UUID           string    `json:"id" db:"uuid"`
	EventID        string    `json:"event_id" db:"tournament_id"`
	ArcherID       string    `json:"archer_id" db:"archer_id"`
	RegistrationID string    `json:"registration_id" db:"registration_id"`
	CertificateNo  string    `json:"certificate_no" db:"certificate_no"`
	IssueDate      time.Time `json:"issue_date" db:"issue_date"`
	PDFURL         *string   `json:"pdf_url" db:"pdf_url"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// SaveCertificateTemplateRequest request payload for updating template
type SaveCertificateTemplateRequest struct {
	HTMLTemplate  string `json:"html_template"`
	BackgroundURL string `json:"background_url"`
	SignatureURL  string `json:"signature_url"`
}

// CertificateItemResponse formatted response for archer earned certificates
type CertificateItemResponse struct {
	ID             string    `json:"id" db:"id"`
	EventID        string    `json:"event_id" db:"event_id"`
	EventSlug      string    `json:"event_slug" db:"event_slug"`
	EventName      string    `json:"event_name" db:"event_name"`
	EventBanner    *string   `json:"event_banner" db:"event_banner"`
	CategoryName   string    `json:"category_name" db:"category_name"`
	ArcherName     string    `json:"archer_name" db:"archer_name"`
	Title          string    `json:"title" db:"title"`
	CertificateNo  string    `json:"certificate_no" db:"certificate_no"`
	IssueDate      time.Time `json:"issue_date" db:"issue_date"`
	PDFURL         string    `json:"pdf_url" db:"pdf_url"`
	VerificationURL string   `json:"verification_url" db:"verification_url"`
}

// VerifyCertificateResponse public verification details
type VerifyCertificateResponse struct {
	Valid          bool      `json:"valid"`
	CertificateNo  string    `json:"certificate_no"`
	ArcherName     string    `json:"archer_name"`
	EventName      string    `json:"event_name"`
	CategoryName   string    `json:"category_name"`
	OrganizerName  string    `json:"organizer_name"`
	IssueDate      time.Time `json:"issue_date"`
	Status         string    `json:"status"`
}
