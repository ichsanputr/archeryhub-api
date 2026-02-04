package models

import "time"

// Team represents a team in an event category
type Team struct {
	UUID         string    `json:"id" db:"uuid"`
	EventID      string    `json:"event_id" db:"tournament_id"`
	CategoryID   string    `json:"category_id" db:"event_id"`
	TeamName     string    `json:"team_name" db:"team_name"`
	CountryCode  string    `json:"country_code" db:"country_code"`
	CountryName  *string   `json:"country_name" db:"country_name"`
	TeamRank     *int      `json:"team_rank" db:"team_rank"`
	TotalScore   int       `json:"total_score" db:"total_score"`
	TotalXCount  int       `json:"total_x_count" db:"total_x_count"`
	Status       string    `json:"status" db:"status"` // active, eliminated, qualified
	MemberCount  int       `json:"member_count" db:"member_count"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// TeamMember represents an individual archer in a team
type TeamMember struct {
	UUID          string  `json:"id" db:"uuid"`
	TeamID        string  `json:"team_id" db:"team_id"`
	ParticipantID string  `json:"participant_id" db:"participant_id"`
	MemberOrder   int     `json:"member_order" db:"member_order"` // 1, 2, 3 for first, second, third
	IsSubstitute  bool    `json:"is_substitute" db:"is_substitute"`
	TotalScore    int     `json:"total_score" db:"total_score"`
	TotalXCount   int     `json:"total_x_count" db:"total_x_count"`
}

// TeamMemberWithDetails includes archer info
type TeamMemberWithDetails struct {
	TeamMember
	FullName    string  `json:"full_name" db:"full_name"`
	BackNumber  *string `json:"back_number" db:"back_number"`
	City        *string `json:"city" db:"city"`
}

// TeamWithMembers includes team members
type TeamWithMembers struct {
	Team
	Members []TeamMemberWithDetails `json:"members"`
}

// TeamScore represents a team's score for a round
type TeamScore struct {
	UUID         string  `json:"id" db:"uuid"`
	TeamID       string  `json:"team_id" db:"team_id"`
	EventID      string  `json:"event_id" db:"tournament_id"`
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
	CategoryID  string   `json:"category_id" binding:"required"`
	TeamName    string   `json:"team_name" binding:"required"`
	CountryCode string   `json:"country_code" binding:"required"`
	CountryName *string  `json:"country_name"`
	MemberIDs   []string `json:"member_ids" binding:"required,min=2,max=4"` // Participant IDs
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
