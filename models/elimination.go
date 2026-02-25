package models

import "time"

// TargetBoardElimination represents a unique code for a target board in an elimination bracket
type TargetBoardElimination struct {
	UUID         string    `json:"id" db:"uuid"`
	BracketUUID  string    `json:"bracket_id" db:"bracket_uuid"`
	CategoryUUID string    `json:"category_id" db:"category_uuid"`
	BoardNumber  int       `json:"board_number" db:"board_number"`
	Code         string    `json:"code" db:"code"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
