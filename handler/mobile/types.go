package mobile

// MobileUser represents user information in mobile login response.
type MobileUser struct {
	UUID      string `json:"uuid"`
	ID        string `json:"id"`
	Username  string `json:"username"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	Role      string `json:"role"`
	UserType  string `json:"user_type"`
}

// MobileLoginResponse represents the response body for mobile login.
type MobileLoginResponse struct {
	Token string     `json:"token"`
	User  MobileUser `json:"user"`
}

// MobileEvent represents event information optimized for mobile.
type MobileEvent struct {
	UUID               string  `db:"uuid" json:"uuid"`
	Name               string  `db:"name" json:"name"`
	Location           string  `db:"location" json:"location"`
	StartDate          string  `db:"start_date" json:"start_date"`
	EndDate            string  `db:"end_date" json:"end_date"`
	LogoURL            *string `db:"logo_url" json:"logo_url"`
	BannerURL          *string `db:"banner_url" json:"banner_url"`
	OrganizerName      string  `db:"organizer_name" json:"organizer_name"`
	OrganizerAvatarURL *string `db:"organizer_avatar_url" json:"organizer_avatar_url"`
	ParticipantCount   int     `db:"participant_count" json:"participant_count"`
}

// MobileEventsResponse represents the list of events for mobile.
type MobileEventsResponse struct {
	Events     []MobileEvent `json:"events"`
	TotalCount int           `json:"total_count"`
}

// ErrorResponse represents a standard error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// MessageResponse represents a standard success message response.
type MessageResponse struct {
	Message string `json:"message"`
}
