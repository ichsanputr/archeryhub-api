package models

import "time"

type Order struct {
	UUID            string    `json:"id" db:"uuid"`
	SellerID        string    `json:"seller_id" db:"seller_id"`
	BuyerID         string    `json:"buyer_id" db:"buyer_id"`
	TotalAmount     float64   `json:"total_amount" db:"total_amount"`
	Status          string    `json:"status" db:"status"` // pending, processing, shipped, done, cancelled
	PaymentStatus   string    `json:"payment_status" db:"payment_status"` // unpaid, paid, expired, failed
	ShippingAddress *string   `json:"shipping_address" db:"shipping_address"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

type OrderItem struct {
	UUID      string    `json:"id" db:"uuid"`
	OrderID   string    `json:"order_id" db:"order_id"`
	ProductID string    `json:"product_id" db:"product_id"`
	Quantity  int       `json:"quantity" db:"quantity"`
	Price     float64   `json:"price" db:"price"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
