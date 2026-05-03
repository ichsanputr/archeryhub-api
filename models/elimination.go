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

// EliminationMatchEndScore represents scores for a specific end in an elimination match
type EliminationMatchEndScore struct {
	EndNo  int `json:"end_no"`
	ScoreA int `json:"score_a"`
	ScoreB int `json:"score_b"`
}

// EliminationMatch represents a single match in an elimination bracket
type EliminationMatch struct {
	UUID            string                     `db:"uuid" json:"uuid"`
	MatchID         string                     `db:"match_id" json:"id"`
	RoundNo         int                        `db:"round_no" json:"round_no"`
	MatchNo         int                        `db:"match_no" json:"match_no"`
	EntryAUUID      *string                    `db:"entry_a_uuid" json:"entry_a_id"`
	EntryAName      *string                    `db:"entry_a_name" json:"entry_a_name"`
	EntryASeed      *int                       `db:"entry_a_seed" json:"entry_a_seed"`
	EntryBUUID      *string                    `db:"entry_b_uuid" json:"entry_b_id"`
	EntryBName      *string                    `db:"entry_b_name" json:"entry_b_name"`
	EntryBSeed      *int                       `db:"entry_b_seed" json:"entry_b_seed"`
	WinnerEntryUUID *string                    `db:"winner_entry_uuid" json:"winner_entry_id"`
	Status          string                     `db:"status" json:"status"`
	IsBye           bool                       `db:"is_bye" json:"is_bye"`
	TotalScoreA     int                        `json:"total_score_a"`
	TotalScoreB     int                        `json:"total_score_b"`
	SetPointsA      int                        `json:"set_points_a"`
	SetPointsB      int                        `json:"set_points_b"`
	ShootOffA       *string                    `json:"shoot_off_a"`
	ShootOffB       *string                    `json:"shoot_off_b"`
	Ends            []EliminationMatchEndScore `json:"ends"`
}

// EliminationResultsResponse represents the full bracket and matches for elimination
type EliminationResultsResponse struct {
	UUID           string                           `json:"uuid"`
	BracketID      string                           `json:"bracket_id"`
	BracketType    string                           `json:"bracket_type"`
	Format         string                           `json:"format"`
	BracketSize    int                              `json:"bracket_size"`
	EndsPerMatch   int                              `json:"ends_per_match"`
	ArrowsPerEnd   int                              `json:"arrows_per_end"`
	GeneratedAt    *string                          `json:"generated_at"`
	MatchesByRound map[int][]EliminationMatch `json:"matches"`
}
