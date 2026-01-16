package models

import (
	"time"
)

// QualificationScore represents qualification round scoring
type QualificationScore struct {
	ID            string    `json:"id" db:"id"`
	TournamentID  string    `json:"tournament_id" db:"tournament_id"`
	ParticipantID string    `json:"participant_id" db:"participant_id"`
	Session       int       `json:"session" db:"session"`
	DistanceOrder int       `json:"distance_order" db:"distance_order"`
	EndNumber     int       `json:"end_number" db:"end_number"`
	Arrow1        *int      `json:"arrow_1" db:"arrow_1"`
	Arrow2        *int      `json:"arrow_2" db:"arrow_2"`
	Arrow3        *int      `json:"arrow_3" db:"arrow_3"`
	Arrow4        *int      `json:"arrow_4" db:"arrow_4"`
	Arrow5        *int      `json:"arrow_5" db:"arrow_5"`
	Arrow6        *int      `json:"arrow_6" db:"arrow_6"`
	EndTotal      int       `json:"end_total" db:"end_total"`
	RunningTotal  int       `json:"running_total" db:"running_total"`
	XCount        int       `json:"x_count" db:"x_count"`
	TenCount      int       `json:"ten_count" db:"ten_count"`
	Verified      bool      `json:"verified" db:"verified"`
	EnteredBy     *string   `json:"entered_by" db:"entered_by"`
	EnteredAt     time.Time `json:"entered_at" db:"entered_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// SubmitScoreRequest represents the request to submit scores
type SubmitScoreRequest struct {
	ParticipantID string `json:"participant_id" binding:"required"`
	Session       int    `json:"session" binding:"required,min=1"`
	DistanceOrder int    `json:"distance_order" binding:"required,min=1"`
	EndNumber     int    `json:"end_number" binding:"required,min=1"`
	Arrow1        *int   `json:"arrow_1" binding:"omitempty,min=0,max=10"`
	Arrow2        *int   `json:"arrow_2" binding:"omitempty,min=0,max=10"`
	Arrow3        *int   `json:"arrow_3" binding:"omitempty,min=0,max=10"`
	Arrow4        *int   `json:"arrow_4" binding:"omitempty,min=0,max=10"`
	Arrow5        *int   `json:"arrow_5" binding:"omitempty,min=0,max=10"`
	Arrow6        *int   `json:"arrow_6" binding:"omitempty,min=0,max=10"`
}

// ParticipantRanking represents a participant's ranking in qualification
type ParticipantRanking struct {
	ParticipantID string  `json:"participant_id" db:"participant_id"`
	AthleteID     string  `json:"athlete_id" db:"athlete_id"`
	FirstName     string  `json:"first_name" db:"first_name"`
	LastName      string  `json:"last_name" db:"last_name"`
	Country       *string `json:"country" db:"country"`
	Club          *string `json:"club" db:"club"`
	BackNumber    *string `json:"back_number" db:"back_number"`
	TargetNumber  *string `json:"target_number" db:"target_number"`
	TotalScore    int     `json:"total_score" db:"total_score"`
	XCount        int     `json:"x_count" db:"x_count"`
	TenCount      int     `json:"ten_count" db:"ten_count"`
	Rank          int     `json:"rank" db:"rank"`
}

// EliminationMatch represents an elimination round match
type EliminationMatch struct {
	ID              string     `json:"id" db:"id"`
	TournamentID    string     `json:"tournament_id" db:"tournament_id"`
	EventID         string     `json:"event_id" db:"event_id"`
	Round           string     `json:"round" db:"round"` // R32, R16, R8, QF, SF, BM, GM
	MatchNumber     int        `json:"match_number" db:"match_number"`
	Participant1ID  *string    `json:"participant1_id" db:"participant1_id"`
	Participant2ID  *string    `json:"participant2_id" db:"participant2_id"`
	Score1          int        `json:"score1" db:"score1"`
	Score2          int        `json:"score2" db:"score2"`
	SetScore1       int        `json:"set_score1" db:"set_score1"`
	SetScore2       int        `json:"set_score2" db:"set_score2"`
	WinnerID        *string    `json:"winner_id" db:"winner_id"`
	Status          string     `json:"status" db:"status"` // pending, ongoing, completed, bye
	ScheduledTime   *time.Time `json:"scheduled_time" db:"scheduled_time"`
	ActualStartTime *time.Time `json:"actual_start_time" db:"actual_start_time"`
	ActualEndTime   *time.Time `json:"actual_end_time" db:"actual_end_time"`
}

// MatchWithDetails includes participant information
type MatchWithDetails struct {
	EliminationMatch
	Participant1Name    *string `json:"participant1_name" db:"participant1_name"`
	Participant1Country *string `json:"participant1_country" db:"participant1_country"`
	Participant2Name    *string `json:"participant2_name" db:"participant2_name"`
	Participant2Country *string `json:"participant2_country" db:"participant2_country"`
	DivisionName        string  `json:"division_name" db:"division_name"`
	CategoryName        string  `json:"category_name" db:"category_name"`
}

// UpdateMatchScoreRequest represents the request to update match scores
type UpdateMatchScoreRequest struct {
	Score1    *int    `json:"score1" binding:"omitempty,min=0"`
	Score2    *int    `json:"score2" binding:"omitempty,min=0"`
	SetScore1 *int    `json:"set_score1" binding:"omitempty,min=0"`
	SetScore2 *int    `json:"set_score2" binding:"omitempty,min=0"`
	WinnerID  *string `json:"winner_id"`
	Status    *string `json:"status" binding:"omitempty,oneof=pending ongoing completed"`
}

// LiveLeaderboardEntry represents a single entry in the live leaderboard
type LiveLeaderboardEntry struct {
	Rank          int     `json:"rank"`
	ParticipantID string  `json:"participant_id" db:"participant_id"`
	FirstName     string  `json:"first_name" db:"first_name"`
	LastName      string  `json:"last_name" db:"last_name"`
	Country       *string `json:"country" db:"country"`
	BackNumber    *string `json:"back_number" db:"back_number"`
	TotalScore    int     `json:"total_score" db:"total_score"`
	XCount        int     `json:"x_count" db:"x_count"`
	TenCount      int     `json:"ten_count" db:"ten_count"`
	LastEndScore  *int    `json:"last_end_score" db:"last_end_score"`
	EndsCompleted int     `json:"ends_completed" db:"ends_completed"`
	RankChange    int     `json:"rank_change"` // positive = up, negative = down, 0 = no change
}
