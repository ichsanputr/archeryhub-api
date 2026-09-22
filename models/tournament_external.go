package models

import "time"

// TournamentExternal represents an external tournament scraped from Ianseo or other sources
type TournamentExternal struct {
	ID                int64     `json:"id" db:"id"`
	UUID              string    `json:"uuid" db:"uuid"`
	Slug              string    `json:"slug" db:"slug"`
	SourcePlatform    string    `json:"source_platform" db:"source_platform"`
	ExternalID        string    `json:"external_id" db:"external_id"`
	SourceURL         string    `json:"source_url" db:"source_url"`
	Name              string    `json:"name" db:"name"`
	ShortName         *string   `json:"short_name" db:"short_name"`
	Venue             *string   `json:"venue" db:"venue"`
	Location          *string   `json:"location" db:"location"`
	City              *string   `json:"city" db:"city"`
	Country           *string   `json:"country" db:"country"`
	CountryCode       *string   `json:"country_code" db:"country_code"`
	Description       *string   `json:"description" db:"description"`
	StartDate         *string   `json:"start_date" db:"start_date"`
	EndDate           *string   `json:"end_date" db:"end_date"`
	BannerURL         *string   `json:"banner_url" db:"banner_url"`
	LogoURL           *string   `json:"logo_url" db:"logo_url"`
	Status            string    `json:"status" db:"status"`
	CategoriesCount   int       `json:"categories_count" db:"categories_count"`
	ParticipantsCount int       `json:"participants_count" db:"participants_count"`
	DataJSON          string    `json:"data_json" db:"data_json"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// TournamentExternalListItem for listing view
type TournamentExternalListItem struct {
	ID                int64   `json:"id" db:"id"`
	UUID              string  `json:"uuid" db:"uuid"`
	Slug              string  `json:"slug" db:"slug"`
	SourcePlatform    string  `json:"source_platform" db:"source_platform"`
	ExternalID        string  `json:"external_id" db:"external_id"`
	Name              string  `json:"name" db:"name"`
	ShortName         *string `json:"short_name" db:"short_name"`
	Venue             *string `json:"venue" db:"venue"`
	Location          *string `json:"location" db:"location"`
	City              *string `json:"city" db:"city"`
	Country           *string `json:"country" db:"country"`
	CountryCode       *string `json:"country_code" db:"country_code"`
	Description       *string `json:"description" db:"description"`
	StartDate         *string `json:"start_date" db:"start_date"`
	EndDate           *string `json:"end_date" db:"end_date"`
	BannerURL         *string `json:"banner_url" db:"banner_url"`
	LogoURL           *string `json:"logo_url" db:"logo_url"`
	Status            string  `json:"status" db:"status"`
	CategoriesCount   int     `json:"categories_count" db:"categories_count"`
	ParticipantsCount int     `json:"participants_count" db:"participants_count"`
}

// TournamentExternalCategory represents a competition category
type TournamentExternalCategory struct {
	ID             int64     `json:"id" db:"id"`
	TournamentID   int64     `json:"tournament_id" db:"tournament_id"`
	CategoryName   string    `json:"category_name" db:"category_name"`
	CategoryCode   *string   `json:"category_code" db:"category_code"`
	IsTeam         bool      `json:"is_team" db:"is_team"`
	Distance       *string   `json:"distance" db:"distance"`
	TargetFace     *string   `json:"target_face" db:"target_face"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// TournamentExternalAthlete represents an entry / archer
type TournamentExternalAthlete struct {
	ID           int64     `json:"id" db:"id"`
	TournamentID int64     `json:"tournament_id" db:"tournament_id"`
	CategoryID   *int64    `json:"category_id" db:"category_id"`
	CategoryName *string   `json:"category_name" db:"category_name"`
	BIB          *string   `json:"bib" db:"bib"`
	Name         string    `json:"name" db:"name"`
	ClubCode     *string   `json:"club_code" db:"club_code"`
	ClubName     *string   `json:"club_name" db:"club_name"`
	CountryCode  *string   `json:"country_code" db:"country_code"`
	Gender       string    `json:"gender" db:"gender"`
	TargetLane   *string   `json:"target_lane" db:"target_lane"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// TournamentExternalQualification represents qualification scores
type TournamentExternalQualification struct {
	ID           int64   `json:"id" db:"id"`
	TournamentID int64   `json:"tournament_id" db:"tournament_id"`
	CategoryID   int64   `json:"category_id" db:"category_id"`
	CategoryName string  `json:"category_name" db:"category_name"`
	Rank         int     `json:"rank" db:"rank"`
	TargetLane   *string `json:"target_lane" db:"target_lane"`
	BIB          *string `json:"bib" db:"bib"`
	AthleteName  string  `json:"athlete_name" db:"athlete_name"`
	ClubCode     *string `json:"club_code" db:"club_code"`
	ClubName     *string `json:"club_name" db:"club_name"`
	D1Score      int     `json:"d1_score" db:"d1_score"`
	D2Score      int     `json:"d2_score" db:"d2_score"`
	D3Score      int     `json:"d3_score" db:"d3_score"`
	D4Score      int     `json:"d4_score" db:"d4_score"`
	TotalScore   int     `json:"total_score" db:"total_score"`
	TensCount    int     `json:"tens_count" db:"tens_count"`
	XCount       int     `json:"x_count" db:"x_count"`
	IsTeam       bool    `json:"is_team" db:"is_team"`
	TeamMembers  *string `json:"team_members" db:"team_members"`
}

// TournamentExternalMatch represents an elimination bracket match
type TournamentExternalMatch struct {
	ID               int64   `json:"id" db:"id"`
	TournamentID     int64   `json:"tournament_id" db:"tournament_id"`
	CategoryID       int64   `json:"category_id" db:"category_id"`
	CategoryName     string  `json:"category_name" db:"category_name"`
	PhaseName        string  `json:"phase_name" db:"phase_name"`
	MatchOrder       int     `json:"match_order" db:"match_order"`
	IsTeam           bool    `json:"is_team" db:"is_team"`
	Athlete1Name     *string `json:"athlete1_name" db:"athlete1_name"`
	Athlete1Club     *string `json:"athlete1_club" db:"athlete1_club"`
	Athlete1Seed     *int    `json:"athlete1_seed" db:"athlete1_seed"`
	Athlete1Score    *string `json:"athlete1_score" db:"athlete1_score"`
	Athlete1Sets     *string `json:"athlete1_sets" db:"athlete1_sets"`
	Athlete1IsWinner bool    `json:"athlete1_is_winner" db:"athlete1_is_winner"`
	Athlete2Name     *string `json:"athlete2_name" db:"athlete2_name"`
	Athlete2Club     *string `json:"athlete2_club" db:"athlete2_club"`
	Athlete2Seed     *int    `json:"athlete2_seed" db:"athlete2_seed"`
	Athlete2Score    *string `json:"athlete2_score" db:"athlete2_score"`
	Athlete2Sets     *string `json:"athlete2_sets" db:"athlete2_sets"`
	Athlete2IsWinner bool    `json:"athlete2_is_winner" db:"athlete2_is_winner"`
	WinnerName       *string `json:"winner_name" db:"winner_name"`
	WinnerClub       *string `json:"winner_club" db:"winner_club"`
	Status           string  `json:"status" db:"status"`
}

// TournamentExternalFinalRank represents final rankings and medals
type TournamentExternalFinalRank struct {
	ID              int64   `json:"id" db:"id"`
	TournamentID    int64   `json:"tournament_id" db:"tournament_id"`
	CategoryID      int64   `json:"category_id" db:"category_id"`
	CategoryName    string  `json:"category_name" db:"category_name"`
	Rank            int     `json:"rank" db:"rank"`
	ParticipantName string  `json:"participant_name" db:"participant_name"`
	ClubCode        *string `json:"club_code" db:"club_code"`
	ClubName        *string `json:"club_name" db:"club_name"`
	Medal           string  `json:"medal" db:"medal"`
	IsTeam          bool    `json:"is_team" db:"is_team"`
}

// TournamentExternalDocument represents PDF documents attached to tournament
type TournamentExternalDocument struct {
	ID           int64   `json:"id" db:"id"`
	TournamentID int64   `json:"tournament_id" db:"tournament_id"`
	CategoryName *string `json:"category_name" db:"category_name"`
	DocType      string  `json:"doc_type" db:"doc_type"`
	Title        string  `json:"title" db:"title"`
	Filename     string  `json:"filename" db:"filename"`
	FileURL      string  `json:"file_url" db:"file_url"`
}

// TournamentExternalSchedule represents tournament schedule item
type TournamentExternalSchedule struct {
	ID           int64   `json:"id" db:"id"`
	TournamentID int64   `json:"tournament_id" db:"tournament_id"`
	EventDate    *string `json:"event_date" db:"event_date"`
	TimeRange    *string `json:"time_range" db:"time_range"`
	Activity     string  `json:"activity" db:"activity"`
	Stage        *string `json:"stage" db:"stage"`
	SortOrder    int     `json:"sort_order" db:"sort_order"`
}

