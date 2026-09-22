package models

import "time"

// Team represents a team in an event category
type Team struct {
	UUID          string    `json:"id" db:"uuid"`
	TournamentID  string    `json:"tournament_id" db:"tournament_id"`
	EventID       string    `json:"event_id" db:"event_id"` // Legacy category UUID column
	CategoryID    *string   `json:"category_id" db:"category_id"`
	TeamName      string    `json:"team_name" db:"team_name"`
	TeamRank      *int      `json:"team_rank" db:"team_rank"`
	TotalScore    int       `json:"total_score" db:"total_score"`
	TotalXCount   int       `json:"total_x_count" db:"total_x_count"`
	Status        string    `json:"status" db:"status"` // active, eliminated, qualified
	MemberCount   int       `json:"member_count" db:"member_count"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// TeamMember represents an individual archer in a team
type TeamMember struct {
	UUID          string  `json:"id" db:"uuid"`
	TeamID        string  `json:"team_id" db:"team_id"`
	ParticipantID string  `json:"participant_id" db:"participant_id"`
	MemberOrder   int     `json:"member_order" db:"member_order"` // 1, 2, 3 for first, second, third
	TotalScore    int     `json:"total_score" db:"total_score"`
	TotalXCount   int     `json:"total_x_count" db:"total_x_count"`
}

// TeamMemberWithDetails includes archer info
type TeamMemberWithDetails struct {
	TeamMember
	FullName    string  `json:"full_name" db:"full_name"`
	Gender      string  `json:"gender" db:"gender"`
	ClubName    string  `json:"club_name" db:"club_name"`
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
	UUID          string  `json:"id" db:"uuid"`
	TeamID        string  `json:"team_id" db:"team_id"`
	TournamentID  string  `json:"tournament_id" db:"tournament_id"`
	Session       int     `json:"session" db:"session"`
	DistanceOrder int     `json:"distance_order" db:"distance_order"`
	EndNumber     int     `json:"end_number" db:"end_number"`
	MemberScores  string  `json:"member_scores" db:"member_scores"` // JSON array of individual scores
	EndTotal      int     `json:"end_total" db:"end_total"`
	XCount        int     `json:"x_count" db:"x_count"`
	RunningTotal  int     `json:"running_total" db:"running_total"`
	Verified      bool    `json:"verified" db:"verified"`
	EnteredBy     *string `json:"entered_by" db:"entered_by"`
	EnteredAt     string  `json:"entered_at" db:"entered_at"`
}

// CreateTeamRequest for creating a new team
type CreateTeamRequest struct {
	CategoryID  string   `json:"category_id" binding:"required"`
	TeamName    string   `json:"team_name" binding:"required"`
	MemberIDs   []string `json:"member_ids" binding:"required,min=2,max=4"` // Participant IDs
}

// TeamRanking for qualification rankings
type TeamRanking struct {
	Rank        int    `json:"rank" db:"rank"`
	TeamID      string `json:"team_id" db:"team_id"`
	TeamName    string `json:"team_name" db:"team_name"`
	TotalScore  int    `json:"total_score" db:"total_score"`
	Total10     int    `json:"total_10" db:"total_10"`
	TotalXCount int    `json:"total_x_count" db:"total_x_count"`
}

// EligiblePartner represents an archer eligible to join a team
type EligiblePartner struct {
	ArcherID                      string  `json:"archer_id" db:"archer_id"`
	ParticipantID                 *string `json:"participant_id,omitempty" db:"participant_id"`
	FullName                      string  `json:"full_name" db:"full_name"`
	Gender                        string  `json:"gender" db:"gender"`
	ClubName                      string  `json:"club_name" db:"club_name"`
	AvatarURL                     *string `json:"avatar_url" db:"avatar_url"`
	PaymentStatus                 *string `json:"payment_status,omitempty" db:"payment_status"`
	IsAlreadyRegisteredIndividual bool    `json:"is_already_registered_individual" db:"is_already_registered_individual"`
	IndividualFee                 float64 `json:"individual_fee" db:"individual_fee"`
}

