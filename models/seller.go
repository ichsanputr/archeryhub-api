package models

import (
	"time"
)

// Seller represents a marketplace merchant
type Seller struct {
	UUID        string    `json:"id" db:"uuid"`
	UserID      *string   `json:"user_id" db:"user_id"`
	Slug        string    `json:"slug" db:"slug"`
	Email       string    `json:"email" db:"email"`
	StoreName   string    `json:"store_name" db:"store_name"`
	Description *string   `json:"description" db:"description"`
	AvatarURL   *string   `json:"avatar_url" db:"avatar_url"`
	BannerURL   *string   `json:"banner_url" db:"banner_url"`
	Phone       *string   `json:"phone" db:"phone"`
	Address     *string   `json:"address" db:"address"`
	City        *string   `json:"city" db:"city"`
	Province    *string   `json:"province" db:"province"`
	Role        string    `json:"role" db:"role"`
	IsVerified  bool      `json:"is_verified" db:"is_verified"`
	Rating           float64    `json:"rating" db:"rating"`
	TotalSales       int        `json:"total_sales" db:"total_sales"`
	FollowersCount   int        `json:"followers_count" db:"followers_count"`
	ProductCount     int        `json:"product_count" db:"product_count"`
	ChatResponseRate string     `json:"chat_response_rate" db:"chat_response_rate"`
	ChatResponseTime string     `json:"chat_response_time" db:"chat_response_time"`
	LastActiveAt     *time.Time `json:"last_active_at" db:"last_active_at"`
	Status           string     `json:"status" db:"status"` // pending, active, suspended
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// UpdateSellerRequest represents the payload to update a seller's profile
type UpdateSellerRequest struct {
	StoreName    *string `json:"store_name"`
	Slug         *string `json:"slug"`
	Description  *string `json:"description"`
	AvatarURL    *string `json:"avatar_url"`
	BannerURL    *string `json:"banner_url"`
	Phone        *string `json:"phone"`
	Email        *string `json:"email"`
	Address      *string `json:"address"`
	City         *string `json:"city"`
	Province     *string `json:"province"`
}
