package models

import "time"

type Notification struct {
	ID        int       `db:"id" json:"id"`
	UserID    string    `db:"user_id" json:"user_id"`
	UserRole  string    `db:"user_role" json:"user_role"`
	Type      string    `db:"type" json:"type"` // success, warning, danger, info, default
	Title     string    `db:"title" json:"title"`
	Message   string    `db:"message" json:"message"`
	Link      *string   `db:"link" json:"link,omitempty"`
	IsRead    bool      `db:"is_read" json:"is_read"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type NotificationListResponse struct {
	Notifications []Notification `json:"notifications"`
	UnreadCount   int            `json:"unread_count"`
	Total         int            `json:"total"`
}

type CreateNotificationRequest struct {
	UserID   string  `json:"user_id" binding:"required"`
	UserRole string  `json:"user_role" binding:"required"`
	Type     string  `json:"type"`
	Title    string  `json:"title" binding:"required"`
	Message  string  `json:"message" binding:"required"`
	Link     *string `json:"link"`
}
