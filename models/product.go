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
	Colors         *string   `json:"colors" db:"colors"`                 // JSON array
	Specifications *string   `json:"specifications" db:"specifications"` // JSON object
	Views          int       `json:"views" db:"views"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type EnrichedProduct struct {
	Product
	Seller *Seller `json:"seller,omitempty"`
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
	Colors         []string `json:"colors"`
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
	Colors         []string `json:"colors"`
	Specifications any      `json:"specifications"`
}

// CartItem represents an item in a user's shopping cart
type CartItem struct {
	UUID      string    `json:"id" db:"uuid"`
	UserID    string    `json:"user_id" db:"user_id"`
	ProductID string    `json:"product_id" db:"product_id"`
	Quantity  int       `json:"quantity" db:"quantity"`
	Color     *string   `json:"color" db:"color"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Joined data
	ProductName    string   `json:"product_name" db:"product_name"`
	ProductPrice   float64  `json:"product_price" db:"product_price"`
	ProductSale    *float64 `json:"product_sale_price" db:"product_sale_price"`
	ProductImage   *string  `json:"product_image_url" db:"product_image_url"`
	ProductStock   int      `json:"product_stock" db:"product_stock"`
	SellerName     string   `json:"seller_name" db:"seller_name"`
}

// AddToCartRequest represents the payload to add an item to cart
type AddToCartRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,min=1"`
	Color     string `json:"color"`
}

// UpdateCartItemRequest represents the payload to update cart item quantity
type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" binding:"required,min=1"`
}

// CheckoutRequest represents the payload to checkout a cart
type CheckoutRequest struct {
	Method          string `json:"method" binding:"required"`
	ShippingAddress string `json:"shipping_address" binding:"required"`
}
