package mobile

import (
	"archeryhub-api/handler"
	"archeryhub-api/models"
	"time"
)

// MobileEventCategory represents a category summary in event detail response.
type MobileEventCategory struct {
	UUID             string  `json:"uuid"`
	Name             string  `json:"name"`
	RegistrationFee  float64 `json:"registration_fee"`
	MaxParticipants  *int    `json:"max_participants"`
	ParticipantCount int     `json:"participant_count"`
}

// MobileEventDetailPayload represents event detail payload.
type MobileEventDetailPayload struct {
	UUID                 string  `json:"uuid"`
	Name                 string  `json:"name"`
	Slug                 string  `json:"slug"`
	Location             string  `json:"location"`
	StartDate            string  `json:"start_date"`
	EndDate              string  `json:"end_date"`
	RegistrationStart    *string `json:"registration_start"`
	RegistrationDeadline *string `json:"registration_deadline"`
	LogoURL              *string `json:"logo_url"`
	BannerURL            *string `json:"banner_url"`
	Description          *string `json:"description"`
	Status               string  `json:"status"`
	OrganizerName        string  `json:"organizer_name"`
	OrganizerAvatar      *string `json:"organizer_avatar"`
	ParticipantCount     int     `json:"participant_count"`
}

// MobileEventDetailResponse represents /mobile/events/{slug} response.
type MobileEventDetailResponse struct {
	Event              MobileEventDetailPayload `json:"event"`
	Categories         []MobileEventCategory    `json:"categories"`
	IsRegistered       bool                     `json:"is_registered"`
	RegistrationStatus string                   `json:"registration_status"`
}

// MobileRegistrationItem represents one archer registration row.
type MobileRegistrationItem struct {
	ID               string  `json:"id"`
	CategoryName     string  `json:"category_name"`
	PaymentStatus    string  `json:"payment_status"`
	PaymentAmount    float64 `json:"payment_amount"`
	QRRaw            *string `json:"qr_raw"`
	QRCodeDataURL    *string `json:"qr_code_data_url"`
	PaymentMethod    *string `json:"payment_method"`
	TripayReference  *string `json:"tripay_reference"`
	CheckoutURL      *string `json:"checkout_url"`
	Instructions     *string `json:"instructions"`
	VANumber         *string `json:"va_number"`
	PayCode          *string `json:"pay_code"`
	QRURL            *string `json:"qr_url"`
	RegistrationDate string  `json:"registration_date"`
}

// MobileMyRegistrationResponse represents /mobile/archer/events/{id}/registration.
type MobileMyRegistrationResponse struct {
	EventID       string                   `json:"event_id"`
	Registrations []MobileRegistrationItem `json:"registrations"`
}

// MobileMyEventItem represents an event row for archer my events.
type MobileMyEventItem struct {
	EventUUID        string  `json:"event_uuid"`
	EventName        string  `json:"event_name"`
	EventSlug        string  `json:"event_slug"`
	Location         string  `json:"location"`
	StartDate        string  `json:"start_date"`
	EndDate          string  `json:"end_date"`
	LogoURL          *string `json:"logo_url"`
	QRRaw            *string `json:"qr_raw"`
	QRCodeDataURL    *string `json:"qr_code_data_url"`
	CategoryName     string  `json:"category_name"`
	PaymentStatus    string  `json:"payment_status"`
	RegistrationDate string  `json:"registration_date"`
}

// MobileMyEventsResponse represents /mobile/archer/events.
type MobileMyEventsResponse struct {
	Events []MobileMyEventItem `json:"events"`
	Total  int                 `json:"total"`
}

// MobileEventQRCodeResponse represents /mobile/archer/events/{id}/qr.
type MobileEventQRCodeResponse struct {
	EventID          string  `json:"event_id"`
	QRRaw            *string `json:"qr_raw"`
	QRCodeDataURL    *string `json:"qr_code_data_url"`
	PaymentStatus    string  `json:"payment_status"`
	RegistrationDate string  `json:"registration_date"`
}

// MobileNewsListResponse represents /mobile/news list response.
type MobileNewsListResponse struct {
	News       []MobileNewsItem `json:"news"`
	TotalCount int              `json:"total_count"`
}

// MobileNewsDetailResponse represents /mobile/news/{id} response.
type MobileNewsDetailResponse struct {
	News MobileNewsDetail `json:"news"`
}

// MobileMarketplaceProductsResponse represents /mobile/marketplace/products.
type MobileMarketplaceProductsResponse struct {
	Products   []models.Product `json:"products"`
	TotalCount int              `json:"total_count"`
}

// MobileMarketplaceProductResponse represents /mobile/marketplace/products/{id}.
type MobileMarketplaceProductResponse struct {
	Product models.Product `json:"product"`
}

// MobileCartResponse represents /mobile/archer/cart.
type MobileCartResponse struct {
	Data []models.CartItem `json:"data"`
}

// MobileCheckoutResponse represents /mobile/archer/cart/checkout.
type MobileCheckoutResponse struct {
	Message   string      `json:"message"`
	Reference string      `json:"reference"`
	Payment   interface{} `json:"payment"`
}

// MobilePaymentTransactionResponse represents participant payment transaction payload.
type MobilePaymentTransactionResponse struct {
	ID              string  `json:"id"`
	Reference       string  `json:"reference"`
	TripayReference *string `json:"tripay_reference"`
	UserID          string  `json:"user_id"`
	EventID         *string `json:"event_id"`
	RegistrationID  *string `json:"registration_id"`
	Amount          float64 `json:"amount"`
	FeeAmount       float64 `json:"fee_amount"`
	TotalAmount     float64 `json:"total_amount"`
	PaymentMethod   *string `json:"payment_method"`
	VANumber        *string `json:"va_number"`
	QRURL           *string `json:"qr_url"`
	CheckoutURL     *string `json:"checkout_url"`
	PayCode         *string `json:"pay_code"`
	Instructions    *string `json:"instructions"`
	Status          string  `json:"status"`
}

// MobileOrderHistoryItem represents one row in order history.
type MobileOrderHistoryItem struct {
	ID               string  `json:"id"`
	SellerID         string  `json:"seller_id"`
	SellerName       string  `json:"seller_name"`
	TotalAmount      float64 `json:"total_amount"`
	Status           string  `json:"status"`
	PaymentStatus    string  `json:"payment_status"`
	TotalItems       int     `json:"total_items"`
	PaymentReference *string `json:"payment_reference"`
	CheckoutURL      *string `json:"checkout_url"`
	PaymentMethod    *string `json:"payment_method"`
	CreatedAt        string  `json:"created_at"`
}

// MobileArcherOrdersResponse represents /mobile/archer/orders.
type MobileArcherOrdersResponse struct {
	Orders []MobileOrderHistoryItem `json:"orders"`
	Total  int                      `json:"total"`
	Limit  int                      `json:"limit"`
	Offset int                      `json:"offset"`
}

// MobileChatListResponse represents /mobile/chat/conversations.
type MobileChatListResponse struct {
	Conversations []handler.ChatConversation `json:"conversations"`
	UnreadTotal   int                        `json:"unread_total"`
}

// MobileChatMessagesResponse represents /mobile/chat/conversations/{id}/messages.
type MobileChatMessagesResponse struct {
	Conversation *handler.ChatConversation `json:"conversation"`
	Messages     []handler.ChatMessage     `json:"messages"`
}

// MobileChatUnreadResponse represents /mobile/chat/unread.
type MobileChatUnreadResponse struct {
	Unread int `json:"unread"`
}

// MobileChatLastActiveResponse represents /mobile/chat/last-active.
type MobileChatLastActiveResponse struct {
	PeerType   string     `json:"peer_type"`
	PeerID     string     `json:"peer_id"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	IsOnline   bool       `json:"is_online"`
	ServerTime string     `json:"server_time"`
}

// MobileScoringSessionResponse represents scoring session info.
type MobileScoringSessionResponse struct {
	UUID         string `json:"uuid"`
	Name         string `json:"name"`
	EventUUID    string `json:"event_uuid"`
	EventName    string `json:"event_name"`
	TotalEnds    int    `json:"total_ends"`
	ArrowsPerEnd int    `json:"arrows_per_end"`
}

// MobileScoringArcherBoardItem represents archer row in a board/session.
type MobileScoringArcherBoardItem struct {
	AssignmentUUID  string  `json:"assignment_uuid"`
	ParticipantUUID string  `json:"participant_uuid"`
	TargetName      string  `json:"target_name"`
	BoardNumber     int     `json:"board_number"`
	Position        string  `json:"position"`
	Name            string  `json:"name"`
	Division        string  `json:"division"`
	AvatarURL       *string `json:"avatar_url"`
	CurrentScore    int     `json:"current_score"`
	EndsCompleted   int     `json:"ends_completed"`
	LastEndScore    int     `json:"last_end_score"`
	Rank            int     `json:"rank,omitempty"`
}

// MobileGetSessionBoardsResponse represents /mobile/sessions/boards.
type MobileGetSessionBoardsResponse struct {
	Session MobileScoringSessionResponse   `json:"session"`
	Archers []MobileScoringArcherBoardItem `json:"archers"`
}

// MobileScoringAssignmentMeta represents assignment meta payload.
type MobileScoringAssignmentMeta struct {
	UUID         string  `json:"uuid"`
	SessionUUID  string  `json:"session_uuid"`
	SessionName  string  `json:"session_name"`
	EventName    string  `json:"event_name"`
	TargetName   string  `json:"target_name"`
	ArcherName   string  `json:"archer_name"`
	Division     string  `json:"division"`
	AvatarURL    *string `json:"avatar_url"`
	TotalEnds    int     `json:"total_ends"`
	ArrowsPerEnd int     `json:"arrows_per_end"`
}

// MobileScoringSummary represents assignment score summary.
type MobileScoringSummary struct {
	TotalScore    int `json:"total_score"`
	TotalX        int `json:"total_x"`
	TotalTenPlusX int `json:"total_ten_plus_x"`
	EndsCompleted int `json:"ends_completed"`
}

// MobileScoringArrowScore represents one arrow score.
type MobileScoringArrowScore struct {
	ArrowNumber int  `json:"arrow_number"`
	Score       int  `json:"score"`
	IsX         bool `json:"is_x"`
}

// MobileScoringEndScore represents one end score row.
type MobileScoringEndScore struct {
	EndNumber       int                       `json:"end_number"`
	EndScoreUUID    string                    `json:"end_score_uuid"`
	EndTotal        int                       `json:"end_total"`
	XCount          int                       `json:"x_count"`
	TenCount        int                       `json:"ten_count"`
	CumulativeTotal int                       `json:"cumulative_total"`
	Arrows          []MobileScoringArrowScore `json:"arrows"`
}

// MobileAssignmentScoreDetailResponse represents /mobile/assignments/{assignmentId}/detail.
type MobileAssignmentScoreDetailResponse struct {
	Assignment MobileScoringAssignmentMeta `json:"assignment"`
	Summary    MobileScoringSummary        `json:"summary"`
	Ends       []MobileScoringEndScore     `json:"ends"`
}
