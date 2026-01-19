package models

import (
	"encoding/json"
	"strconv"
	"time"
)

// FlexibleFloat64 handles both string and float64 during JSON unmarshaling
type FlexibleFloat64 float64

func (ff *FlexibleFloat64) UnmarshalJSON(data []byte) error {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		*ff = FlexibleFloat64(val)
		return nil
	}
	var val float64
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}
	*ff = FlexibleFloat64(val)
	return nil
}

// PaymentTransaction represents a payment transaction with Tripay
type PaymentTransaction struct {
	ID               string          `json:"id" db:"id"`
	Reference        string          `json:"reference" db:"reference"`
	TripayReference  *string         `json:"tripay_reference" db:"tripay_reference"`
	UserID           string          `json:"user_id" db:"user_id"`
	EventID          *string         `json:"event_id" db:"event_id"`
	RegistrationID   *string         `json:"registration_id" db:"registration_id"`
	Amount           float64         `json:"amount" db:"amount"`
	FeeAmount        float64         `json:"fee_amount" db:"fee_amount"`
	TotalAmount      float64         `json:"total_amount" db:"total_amount"`
	PaymentMethod    *string         `json:"payment_method" db:"payment_method"`
	PaymentChannel   *string         `json:"payment_channel" db:"payment_channel"`
	VANumber         *string         `json:"va_number" db:"va_number"`
	QRURL            *string         `json:"qr_url" db:"qr_url"`
	CheckoutURL      *string         `json:"checkout_url" db:"checkout_url"`
	PayCode          *string         `json:"pay_code" db:"pay_code"`
	Instructions     *string         `json:"instructions" db:"instructions"`
	Status           string          `json:"status" db:"status"` // pending, paid, expired, failed, refunded
	PaidAt           *time.Time      `json:"paid_at" db:"paid_at"`
	ExpiredAt        time.Time       `json:"expired_at" db:"expired_at"`
	CallbackData     json.RawMessage `json:"callback_data" db:"callback_data"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`
}

// CreatePaymentRequest represents the request to create a payment
type CreatePaymentRequest struct {
	Method         string  `json:"method" binding:"required"` // Payment channel code (e.g., BRIVA, QRIS)
	EventID        string  `json:"event_id" binding:"required"`
	RegistrationID *string `json:"registration_id"`
	Type           string  `json:"type"` // e.g., "registration" (default) or "platform_fee"
}

// PaymentChannelFee represents the fee details for a Tripay channel
type PaymentChannelFee struct {
	Flat    int             `json:"flat"`
	Percent FlexibleFloat64 `json:"percent"`
}

// PaymentChannel represents a Tripay payment channel
type PaymentChannel struct {
	Group         string            `json:"group"`
	Code          string            `json:"code"`
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	FeeMerchant   PaymentChannelFee `json:"fee_merchant"`
	FeeCustomer   PaymentChannelFee `json:"fee_customer"`
	TotalFee      PaymentChannelFee `json:"total_fee"`
	MinimumFee    int               `json:"minimum_fee"`
	MaximumFee    int               `json:"maximum_fee"`
	MinimumAmount int               `json:"minimum_amount"`
	MaximumAmount int               `json:"maximum_amount"`
	IconURL       string            `json:"icon_url"`
	Active        bool              `json:"active"`
}

// EventRegistration represents a user's registration for a event
type EventRegistration struct {
	ID                 string     `json:"id" db:"id"`
	EventID            string     `json:"event_id" db:"event_id"`
	UserID             string     `json:"user_id" db:"user_id"`
	AthleteName        string     `json:"athlete_name" db:"athlete_name"`
	AthleteEmail       *string    `json:"athlete_email" db:"athlete_email"`
	AthletePhone       *string    `json:"athlete_phone" db:"athlete_phone"`
	ClubName           *string    `json:"club_name" db:"club_name"`
	Division           string     `json:"division" db:"division"`
	Category           string     `json:"category" db:"category"`
	BowType            string     `json:"bow_type" db:"bow_type"`
	EntryFee           float64    `json:"entry_fee" db:"entry_fee"`
	AdminFee           float64    `json:"admin_fee" db:"admin_fee"`
	TotalFee           float64    `json:"total_fee" db:"total_fee"`
	PaymentStatus      string     `json:"payment_status" db:"payment_status"` // unpaid, pending, paid, refunded
	PaymentID          *string    `json:"payment_id" db:"payment_id"`
	RegistrationNumber *string    `json:"registration_number" db:"registration_number"`
	Status             string     `json:"status" db:"status"` // pending, approved, rejected, cancelled
	Notes              *string    `json:"notes" db:"notes"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

// RegisterEventRequest represents the request to register for a event
type RegisterEventRequest struct {
	AthleteName  string  `json:"athlete_name" binding:"required"`
	AthleteEmail *string `json:"athlete_email"`
	AthletePhone *string `json:"athlete_phone"`
	ClubName     *string `json:"club_name"`
	Division     string  `json:"division" binding:"required"`
	Category     string  `json:"category" binding:"required"`
	BowType      string  `json:"bow_type" binding:"required"`
}
