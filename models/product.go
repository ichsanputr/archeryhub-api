package models

import (
	"time"
)

// Product represents a marketplace item
type Product struct {
	UUID           string    `json:"id" db:"uuid"`
	OrganizationID *string   `json:"organization_id" db:"organization_id"`
	ClubID         *string   `json:"club_id" db:"club_id"`
	SellerID       *string   `json:"seller_id" db:"seller_id"`
	Name           string    `json:"name" db:"name"`
	Slug           string    `json:"slug" db:"slug"`
	Description    *string   `json:"description" db:"description"`
	Price          float64   `json:"price" db:"price"`
	SalePrice      *float64  `json:"sale_price" db:"sale_price"`
	Category       string    `json:"category" db:"category"` // equipment, apparel, accessories, training, other
	Stock          int       `json:"stock" db:"stock"`
	Status         string    `json:"status" db:"status"` // draft, active, sold_out, archived
	ImageURL       *string   `json:"image_url" db:"image_url"`
	Images         *string   `json:"images" db:"images"`                 // JSON array
	Specifications *string   `json:"specifications" db:"specifications"` // JSON object
	Views          int       `json:"views" db:"views"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// Seller represents a marketplace merchant (could be organization, club, or individual)
type Seller struct {
	UUID        string    `json:"id" db:"uuid"`
	UserID      *string   `json:"user_id" db:"user_id"`
	StoreName   string    `json:"store_name" db:"store_name"`
	StoreSlug   string    `json:"store_slug" db:"store_slug"`
	Description *string   `json:"description" db:"description"`
	LogoURL     *string   `json:"logo_url" db:"logo_url"`
	BannerURL   *string   `json:"banner_url" db:"banner_url"`
	Phone       *string   `json:"phone" db:"phone"`
	Email       *string   `json:"email" db:"email"`
	Address     *string   `json:"address" db:"address"`
	City        *string   `json:"city" db:"city"`
	Province    *string   `json:"province" db:"province"`
	IsVerified  bool      `json:"is_verified" db:"is_verified"`
	Rating      float64   `json:"rating" db:"rating"`
	TotalSales  int       `json:"total_sales" db:"total_sales"`
	Status      string    `json:"status" db:"status"` // pending, active, suspended
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CreateProductRequest represents the payload to create a new product
type CreateProductRequest struct {
	Name           string   `json:"name" binding:"required"`
	Description    *string  `json:"description"`
	Price          float64  `json:"price" binding:"required"`
	SalePrice      *float64 `json:"sale_price"`
	Category       string   `json:"category" binding:"required"`
	Stock          int      `json:"stock"`
	Status         string   `json:"status"`
	ImageURL       *string  `json:"image_url"`
	Images         []string `json:"images"`
	Specifications any      `json:"specifications"`
}

// UpdateProductRequest represents the payload to update an existing product
type UpdateProductRequest struct {
	Name           *string  `json:"name"`
	Description    *string  `json:"description"`
	Price          *float64 `json:"price"`
	SalePrice      *float64 `json:"sale_price"`
	Category       *string  `json:"category"`
	Stock          *int     `json:"stock"`
	Status         *string  `json:"status"`
	ImageURL       *string  `json:"image_url"`
	Images         []string `json:"images"`
	Specifications any      `json:"specifications"`
}
