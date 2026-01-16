package models

import "time"

// Team represents a team in a tournament event
type Team struct {
	ID           string    `json:"id" db:"id"`
	TournamentID string    `json:"tournament_id" db:"tournament_id"`
	EventID      string    `json:"event_id" db:"event_id"`
	TeamName     string    `json:"team_name" db:"team_name"`
	CountryCode  string    `json:"country_code" db:"country_code"`
	CountryName  *string   `json:"country_name" db:"country_name"`
	TeamRank     *int      `json:"team_rank" db:"team_rank"`
	TotalScore   int       `json:"total_score" db:"total_score"`
	TotalXCount  int       `json:"total_x_count" db:"total_x_count"`
	Status       string    `json:"status" db:"status"` // active, eliminated, qualified
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// TeamMember represents an individual athlete in a team
type TeamMember struct {
	ID            string  `json:"id" db:"id"`
	TeamID        string  `json:"team_id" db:"team_id"`
	ParticipantID string  `json:"participant_id" db:"participant_id"`
	MemberOrder   int     `json:"member_order" db:"member_order"` // 1, 2, 3 for first, second, third
	IsSubstitute  bool    `json:"is_substitute" db:"is_substitute"`
	TotalScore    int     `json:"total_score" db:"total_score"`
	TotalXCount   int     `json:"total_x_count" db:"total_x_count"`
}

// TeamMemberWithDetails includes athlete info
type TeamMemberWithDetails struct {
	TeamMember
	FirstName   string  `json:"first_name" db:"first_name"`
	LastName    string  `json:"last_name" db:"last_name"`
	BackNumber  *string `json:"back_number" db:"back_number"`
	Country     *string `json:"country" db:"country"`
}

// TeamWithMembers includes team members
type TeamWithMembers struct {
	Team
	Members []TeamMemberWithDetails `json:"members"`
}

// TeamScore represents a team's score for a round
type TeamScore struct {
	ID           string  `json:"id" db:"id"`
	TeamID       string  `json:"team_id" db:"team_id"`
	TournamentID string  `json:"tournament_id" db:"tournament_id"`
	Session      int     `json:"session" db:"session"`
	DistanceOrder int    `json:"distance_order" db:"distance_order"`
	EndNumber    int     `json:"end_number" db:"end_number"`
	MemberScores string  `json:"member_scores" db:"member_scores"` // JSON array of individual scores
	EndTotal     int     `json:"end_total" db:"end_total"`
	XCount       int     `json:"x_count" db:"x_count"`
	RunningTotal int     `json:"running_total" db:"running_total"`
	Verified     bool    `json:"verified" db:"verified"`
	EnteredBy    *string `json:"entered_by" db:"entered_by"`
	EnteredAt    string  `json:"entered_at" db:"entered_at"`
}

// CreateTeamRequest for creating a new team
type CreateTeamRequest struct {
	EventID     string   `json:"event_id" binding:"required"`
	TeamName    string   `json:"team_name" binding:"required"`
	CountryCode string   `json:"country_code" binding:"required"`
	CountryName *string  `json:"country_name"`
	MemberIDs   []string `json:"member_ids" binding:"required,min=2,max=4"` // Participant IDs
}

// TeamEliminationMatch represents a team match in elimination rounds
type TeamEliminationMatch struct {
	ID              string  `json:"id" db:"id"`
	TournamentID    string  `json:"tournament_id" db:"tournament_id"`
	EventID         string  `json:"event_id" db:"event_id"`
	Round           string  `json:"round" db:"round"` // R8, QF, SF, BM, GM
	MatchNumber     int     `json:"match_number" db:"match_number"`
	Team1ID         *string `json:"team1_id" db:"team1_id"`
	Team2ID         *string `json:"team2_id" db:"team2_id"`
	Score1          int     `json:"score1" db:"score1"`
	Score2          int     `json:"score2" db:"score2"`
	SetScore1       int     `json:"set_score1" db:"set_score1"`
	SetScore2       int     `json:"set_score2" db:"set_score2"`
	WinnerID        *string `json:"winner_id" db:"winner_id"`
	Status          string  `json:"status" db:"status"` // pending, ongoing, completed
	ScheduledTime   *string `json:"scheduled_time" db:"scheduled_time"`
	ActualStartTime *string `json:"actual_start_time" db:"actual_start_time"`
	ActualEndTime   *string `json:"actual_end_time" db:"actual_end_time"`
}

// TeamRanking for qualification rankings
type TeamRanking struct {
	Rank        int    `json:"rank" db:"rank"`
	TeamID      string `json:"team_id" db:"team_id"`
	TeamName    string `json:"team_name" db:"team_name"`
	CountryCode string `json:"country_code" db:"country_code"`
	TotalScore  int    `json:"total_score" db:"total_score"`
	TotalXCount int    `json:"total_x_count" db:"total_x_count"`
}
