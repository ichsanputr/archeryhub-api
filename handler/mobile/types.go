package mobile

import "archeryhub-api/models"
 
// MobileResponse is a generic response for mobile API.
type MobileResponse struct {
	Message string `json:"message" example:"Success"`
	Success bool   `json:"success" example:"true"`
}

// MessageResponse is a simple message response.
type MessageResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

type Intent struct {
	Name     string   `json:"name"`
	Examples []string `json:"examples"`
	Answer   string   `json:"answer"`
}

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
	Token     string     `json:"token"`
	IsNewUser bool       `json:"is_new_user"`
	User      MobileUser `json:"user"`
}

// MobileArcherRegisterRequest represents the registration payload for a new archer.
type MobileArcherRegisterRequest struct {
	Email     string `json:"email" example:"archer@example.com"`
	Password  string `json:"password" example:"securepassword123"`
	FullName  string `json:"full_name" example:"Rizky Pratama"`
	Phone     string `json:"phone" example:"081234567890"`
	Username  string `json:"username" example:"rizky-pratama"`
	Gender    string `json:"gender" example:"male"`
	BirthDate string `json:"birth_date" example:"1995-05-18"`
	AvatarURL string `json:"avatar_url" example:"https://cdn.archeryhub.id/media/archers/rizky.jpg"`
}

// MobileEventRegistrationRequest represents the payload for event registration.
type MobileEventRegistrationRequest struct {
	EventID            string   `json:"event_id" example:"national-open-2026"`
	AthleteID          string   `json:"athlete_id" example:"arc-a49ee7d7-9d7b-4be7-8652-342f2fca23f9"`
	EventCategoryID    string   `json:"event_category_id" example:"cat-recurve-adult-putra"`
	EventCategoryIDs   []string `json:"event_category_ids" example:"[\"cat-recurve-adult-putra\", \"cat-recurve-adult-team\"]"`
	PaymentAmount      float64  `json:"payment_amount" example:"450000"`
	PaymentProofURLs   []string `json:"payment_proof_urls" example:"[\"https://cdn.archeryhub.id/media/proofs/receipt-001.jpg\"]"`
	PaymentStatus      string   `json:"payment_status" example:"menunggu acc"`
	RegistrationSource string   `json:"registration_source" example:"mobile_app"`
	PaymentType        string   `json:"payment_type" example:"manual"` // manual, online, gateway
}

// MobileAddNewsCommentRequest represents the payload for adding a news comment.
type MobileAddNewsCommentRequest struct {
	UserID    string `json:"user_id" example:"arc-a49ee7d7-9d7b-4be7-8652-342f2fca23f9"`
	GuestName string `json:"guest_name" example:"Budi Santoso"`
	Content   string `json:"content" example:"Wah acaranya seru banget!"`
}

// MobileChatbotMessageRequest represents the payload for chatbot interaction.
type MobileChatbotMessageRequest struct {
	Message string `json:"message" binding:"required" example:"Halo, ada event apa ya?"`
}

// MobileCreatePaymentRequest represents the payload to create a payment.
type MobileCreatePaymentRequest struct {
	Method string `json:"method" binding:"required" example:"BRIVA"`
}

type MobileNewsComment struct {
	UUID      string  `db:"uuid" json:"id"`
	NewsID    string  `db:"news_id" json:"news_id"`
	UserID    *string `db:"user_id" json:"user_id,omitempty"`
	UserType  string  `db:"user_type" json:"user_type"`
	GuestName *string `db:"guest_name" json:"guest_name,omitempty"`
	UserName  string  `db:"user_name" json:"user_name"`
	Content   string  `db:"content" json:"content"`
	CreatedAt string  `db:"created_at" json:"created_at"`
}

type MobileNewsItem struct {
	UUID        string  `db:"uuid" json:"id"`
	Title       string  `db:"title" json:"title"`
	Slug        string  `db:"slug" json:"slug"`
	Excerpt     *string `db:"excerpt" json:"excerpt,omitempty"`
	ImageURL    *string `db:"image_url" json:"image_url,omitempty"`
	Category    string  `db:"category" json:"category"`
	AuthorName  *string `db:"author_name" json:"author_name,omitempty"`
	Views       int     `db:"views" json:"views"`
	PublishedAt *string `db:"published_at" json:"published_at,omitempty"`
	CreatedAt   string  `db:"created_at" json:"created_at"`
}

type MobileNewsDetail struct {
	UUID            string  `db:"uuid" json:"id"`
	Title           string  `db:"title" json:"title"`
	Slug            string  `db:"slug" json:"slug"`
	Excerpt         *string `db:"excerpt" json:"excerpt,omitempty"`
	Content         *string `db:"content" json:"content,omitempty"`
	ImageURL        *string `db:"image_url" json:"image_url,omitempty"`
	Category        string  `db:"category" json:"category"`
	Tags            *string `db:"tags" json:"tags,omitempty"`
	AuthorName      *string `db:"author_name" json:"author_name,omitempty"`
	MetaTitle       *string `db:"meta_title" json:"meta_title,omitempty"`
	MetaDescription *string `db:"meta_description" json:"meta_description,omitempty"`
	Views           int     `db:"views" json:"views"`
	PublishedAt     *string `db:"published_at" json:"published_at,omitempty"`
	CreatedAt       string  `db:"created_at" json:"created_at"`
	UpdatedAt       string  `db:"updated_at" json:"updated_at"`
}

// MobileEvent represents event information optimized for mobile.
type MobileEvent struct {
	UUID               string  `db:"uuid" json:"uuid"`
	Slug               string  `db:"slug" json:"slug"`
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

// MobileEventDetail represents the slimmed-down core info for an event.
type MobileEventDetail struct {
	models.Event
	OrganizerName      *string                    `db:"organizer_name" json:"organizer_name"`
	OrganizerAvatarURL *string                    `db:"organizer_avatar_url" json:"organizer_avatar_url"`
	OrganizerSlug      *string                    `db:"organizer_slug" json:"organizer_slug"`
	OrganizerPhone     *string                    `db:"organizer_phone" json:"organizer_phone"`
	ParticipantCount   int                        `db:"participant_count" json:"participant_count"`
	LocationDetail     models.EventLocationDetail `db:"-" json:"location_detail"`
}

// MobileEventsResponse represents the list of events for mobile.
type MobileEventsResponse struct {
	Events     []MobileEvent `json:"events"`
	TotalCount int           `json:"total_count"`
}

// MobileRegistrationItem represents one archer registration row.
type MobileRegistrationItem struct {
	ID               string  `db:"uuid" json:"id"`
	CategoryName     string  `db:"category_name" json:"category_name"`
	PaymentStatus    string  `db:"payment_status" json:"payment_status"`
	PaymentAmount    float64 `db:"payment_amount" json:"payment_amount"`
	QRRaw            *string `db:"qr_raw" json:"qr_raw"`
	QRCodeDataURL    *string `json:"qr_code_data_url"`
	PaymentMethod    *string `db:"payment_method" json:"payment_method"`
	TripayReference  *string `db:"tripay_reference" json:"tripay_reference"`
	CheckoutURL      *string `db:"checkout_url" json:"checkout_url"`
	Instructions     *string `db:"instructions" json:"instructions"`
	VANumber         *string `db:"va_number" json:"va_number"`
	PayCode          *string `db:"pay_code" json:"pay_code"`
	QRURL            *string `db:"qr_url" json:"qr_url"`
	RegistrationDate string  `db:"registration_date" json:"registration_date"`
}

// MobileNewsListResponse represents news list response.
type MobileNewsListResponse struct {
	News       []MobileNewsItem `json:"news"`
	TotalCount int              `json:"total_count"`
}

// MobileNewsDetailResponse represents news detail response.
type MobileNewsDetailResponse struct {
	News MobileNewsDetail `json:"news"`
}

// MobileNewsCommentsResponse represents news comments list response.
type MobileNewsCommentsResponse struct {
	Comments []MobileNewsComment `json:"comments"`
	Count    int                 `json:"count"`
}

// MobileMarketplaceProductsResponse represents marketplace products list.
type MobileMarketplaceProductsResponse struct {
	Products    []models.Product `json:"products"`
	TotalCount  int              `json:"total_count"`
	Limit       int              `json:"limit"`
	Offset      int              `json:"offset"`
	CurrentPage int              `json:"current_page"`
	LastPage    int              `json:"last_page"`
}

// MobileMarketplaceProductResponse represents marketplace product detail.
type MobileMarketplaceProductResponse struct {
	Product models.Product `json:"product"`
}

// MobileRelatedNewsResponse represents related news items.
type MobileRelatedNewsResponse struct {
	News []MobileNewsItem `json:"news"`
}

// MobilePaymentTransactionResponse represents payment details.
type MobilePaymentTransactionResponse struct {
	ID              string  `json:"id"`
	Reference       string  `json:"reference"`
	TripayReference *string `json:"tripay_reference"`
	Amount          float64 `json:"amount"`
	VANumber        *string `json:"va_number"`
	CheckoutURL     *string `json:"checkout_url"`
	Status          string  `json:"status"`
}

// MobileCartResponse represents cart contents.
type MobileCartResponse struct {
	Data interface{} `json:"data"`
}

// MobileCheckoutResponse represents checkout result.
type MobileCheckoutResponse struct {
	Message   string      `json:"message"`
	Reference string      `json:"reference"`
	Payment   interface{} `json:"payment"`
}

// MobileArcherOrdersResponse represents orders for an archer.
type MobileArcherOrdersResponse struct {
	Orders []MobileOrderHistoryItem `json:"orders"`
	Total  int                      `json:"total"`
	Limit  int                      `json:"limit"`
	Offset int                      `json:"offset"`
}

// MobileChatbotResponse represents chatbot message response.
type MobileChatbotResponse struct {
	Intent       string   `json:"intent"`
	Confidence   float64  `json:"confidence"`
	Answer       string   `json:"answer"`
	QuickActions []string `json:"quick_actions"`
}

// MobileChatbotIntentsResponse represents chatbot intents list.
type MobileChatbotIntentsResponse struct {
	Intents []Intent `json:"intents"`
}

// MobileScorekeeperMeResponse represents scorekeeper profile info.
type MobileScorekeeperMeResponse struct {
	UUID             string  `json:"uuid"`
	OrganizationUUID string  `json:"organization_uuid"`
	Code             string  `json:"code"`
	Name             string  `json:"name"`
	Email            *string `json:"email"`
	AvatarURL        *string `json:"avatar_url"`
	Status           string  `json:"status"`
	OrganizationName string  `json:"organization_name"`
}

// MobileScorekeeperEventsResponse represents events for scorekeeper.
type MobileScorekeeperEventsResponse struct {
	Events     []MobileEvent `json:"events"`
	TotalCount int           `json:"total_count"`
}

// MobileScanTargetResponse represents the result of scanning a target QR.
type MobileScanTargetResponse struct {
	Type    string      `json:"type" example:"qualification"`
	Board   interface{} `json:"board"`
	Archers interface{} `json:"archers"`
	Matches interface{} `json:"matches,omitempty"`
}

// MobileSessionBoardsResponse represents the leaderboard for a session.
type MobileSessionBoardsResponse struct {
	Session interface{} `json:"session"`
	Archers interface{} `json:"archers"`
}

// MobileAssignmentScoreDetailResponse represents arrow-by-arrow scores.
type MobileAssignmentScoreDetailResponse struct {
	Assignment interface{} `json:"assignment"`
	Summary    interface{} `json:"summary"`
	Ends       interface{} `json:"ends"`
}

// MobileMyRegistrationResponse represents registrations for a specific event.
type MobileMyRegistrationResponse struct {
	EventID       string                   `json:"event_id"`
	Registrations []MobileRegistrationItem `json:"registrations"`
}

// MobileMyEventsResponse represents events the archer is participating in.
type MobileMyEventsResponse struct {
	Events []MobileMyEventItem `json:"events"`
	Total  int                 `json:"total"`
}


// MobileMyEventItem represents an event row for archer my events.
type MobileMyEventItem struct {
	EventUUID        string  `db:"event_uuid" json:"event_uuid"`
	EventName        string  `db:"event_name" json:"event_name"`
	EventSlug        string  `db:"event_slug" json:"event_slug"`
	Location         *string `db:"location" json:"location"`
	StartDate        *string `db:"start_date" json:"start_date"`
	EndDate          *string `db:"end_date" json:"end_date"`
	LogoURL          *string `db:"logo_url" json:"logo_url"`
	QRRaw            *string `db:"qr_raw" json:"qr_raw"`
	QRCodeDataURL    *string `json:"qr_code_data_url"`
	CategoryName     string  `db:"category_name" json:"category_name"`
	PaymentStatus    string  `db:"payment_status" json:"payment_status"`
	RegistrationDate *string `db:"registration_date" json:"registration_date"`
}

// MobileOrderHistoryItem represents one row in order history.
type MobileOrderHistoryItem struct {
	ID               string  `db:"uuid" json:"id"`
	SellerID         string  `db:"seller_id" json:"seller_id"`
	SellerName       string  `db:"seller_name" json:"seller_name"`
	TotalAmount      float64 `db:"total_amount" json:"total_amount"`
	Status           string  `db:"status" json:"status"`
	PaymentStatus    string  `db:"payment_status" json:"payment_status"`
	TotalItems       int     `db:"total_items" json:"total_items"`
	PaymentReference *string `db:"reference" json:"payment_reference"`
	CheckoutURL      *string `db:"checkout_url" json:"checkout_url"`
	PaymentMethod    *string `db:"payment_method" json:"payment_method"`
	CreatedAt        string  `db:"created_at" json:"created_at"`
}

// MobileRegisterEventResponse represents a success response for event registration.
type MobileRegisterEventResponse struct {
	Message              string   `json:"message"`
	RegistrationID       string   `json:"registration_id"`
	RegisteredCategories []string `json:"registered_categories"`
	PaymentStatus        string   `json:"payment_status"`
}

// ErrorResponse represents a standard error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// MobileSellerProfileData represents seller profile data for mobile.
type MobileSellerProfileData struct {
	ID            string                 `json:"id" example:"sel-7d7e8b16-5a11-4c4f-8f13-1c4f14d97d8e"`
	UUID          string                 `json:"uuid" example:"sel-7d7e8b16-5a11-4c4f-8f13-1c4f14d97d8e"`
	StoreName     string                 `json:"store_name" example:"ArcheryHub Store Jakarta"`
	Slug          *string                `json:"slug" example:"archeryhub-store-jakarta"`
	StoreSlug     *string                `json:"store_slug" example:"archeryhub-store-jakarta"`
	Name          string                 `json:"name" example:"ArcheryHub Store Jakarta"`
	Username      *string                `json:"username" example:"archeryhub-store-jakarta"`
	Description   *string                `json:"description" example:"Toko perlengkapan panahan untuk kebutuhan latihan dan kompetisi."`
	AvatarURL     *string                `json:"avatar_url" example:"https://cdn.archeryhub.id/media/seller/logo-store.png"`
	BannerURL     *string                `json:"banner_url" example:"https://cdn.archeryhub.id/media/seller/banner-store.jpg"`
	Logo          *string                `json:"logo" example:"https://cdn.archeryhub.id/media/seller/logo-store.png"`
	Banner        *string                `json:"banner" example:"https://cdn.archeryhub.id/media/seller/banner-store.jpg"`
	Phone         *string                `json:"phone" example:"081234567890"`
	Email         *string                `json:"email" example:"seller@archeryhub.id"`
	Address       *string                `json:"address" example:"Jl. Panahan No. 10, Jakarta"`
	City          *string                `json:"city" example:"Jakarta"`
	Province      *string                `json:"province" example:"DKI Jakarta"`
	Sections      map[string]interface{} `json:"sections,omitempty" swaggertype:"object"`
	CatalogConfig map[string]interface{} `json:"catalog_config,omitempty" swaggertype:"object"`
	ThemeColor    *string                `json:"theme_color,omitempty" example:"#C1121F"`
	BannerText    *string                `json:"banner_text,omitempty" example:"Perlengkapan panahan lengkap untuk semua level"`
	PageSettings  map[string]interface{} `json:"page_settings,omitempty" swaggertype:"object"`
	UserType      string                 `json:"user_type" example:"seller"`
}

// MobileSellerProfileResponse represents /mobile/seller/me.
type MobileSellerProfileResponse struct {
	Data MobileSellerProfileData `json:"data"`
}

// MobileSellerProductsResponse represents /mobile/seller/products.
type MobileSellerProductsResponse struct {
	Products []models.Product `json:"products"`
	Total    int              `json:"total"`
	Limit    int              `json:"limit"`
	Offset   int              `json:"offset"`
}

// MobileOrganizationProfileData represents organization profile data for mobile.
type MobileOrganizationProfileData struct {
	ID                    string                 `json:"id" example:"org-1fa53fca-b7da-4be6-b6eb-8b962d2f7d55"`
	UUID                  string                 `json:"uuid" example:"org-1fa53fca-b7da-4be6-b6eb-8b962d2f7d55"`
	Slug                  *string                `json:"slug" example:"archeryhub-jakarta"`
	Name                  string                 `json:"name" example:"ArcheryHub Jakarta"`
	Acronym               *string                `json:"acronym" example:"AHJ"`
	Description           *string                `json:"description" example:"Organisasi panahan yang fokus pada pembinaan atlet dan penyelenggaraan event."`
	Website               *string                `json:"website" example:"https://archeryhub.id"`
	Email                 string                 `json:"email" example:"event@archeryhub.id"`
	WhatsAppNo            *string                `json:"whatsapp_no" example:"081234567890"`
	AvatarURL             *string                `json:"avatar_url" example:"https://cdn.archeryhub.id/media/organization/logo.png"`
	BannerURL             *string                `json:"banner_url" example:"https://cdn.archeryhub.id/media/organization/banner.jpg"`
	Address               *string                `json:"address" example:"Jl. Stadion Utama No. 1, Jakarta"`
	City                  *string                `json:"city" example:"Jakarta"`
	Country               *string                `json:"country" example:"Indonesia"`
	RegistrationNumber    *string                `json:"registration_number" example:"AHJ-2026-001"`
	EstablishedDate       *string                `json:"established_date" example:"2020-08-17"`
	ContactPersonName     *string                `json:"contact_person_name" example:"Budi Santoso"`
	ContactPersonEmail    *string                `json:"contact_person_email" example:"budi@archeryhub.id"`
	ContactPersonPhone    *string                `json:"contact_person_phone" example:"081298765432"`
	SocialFacebook        *string                `json:"social_facebook" example:"archeryhubjakarta"`
	SocialInstagram       *string                `json:"social_instagram" example:"archeryhub.jakarta"`
	SocialTwitter         *string                `json:"social_twitter" example:"archeryhubjkt"`
	SocialMedia           interface{}            `json:"social_media,omitempty" swaggertype:"object"`
	Vision                *string                `json:"vision" example:"Menjadi pusat pembinaan panahan modern di Indonesia."`
	Mission               *string                `json:"mission" example:"Membina atlet, pelatih, dan penyelenggaraan kompetisi yang profesional."`
	History               *string                `json:"history" example:"Didirikan untuk mendukung ekosistem panahan nasional yang lebih terstruktur."`
	FAQ                   []interface{}          `json:"faq,omitempty"`
	VerificationStatus    *string                `json:"verification_status" example:"verified"`
	Status                *string                `json:"status" example:"active"`
	CreatedAt             string                 `json:"created_at" example:"2026-01-10T08:00:00Z"`
	UpdatedAt             string                 `json:"updated_at" example:"2026-03-10T09:30:00Z"`
	SubscriptionStatus    string                 `json:"subscription_status" example:"active"`
	SubscriptionExpiresAt *string                `json:"subscription_expires_at" example:"2026-12-31T23:59:59Z"`
	PageSettings          map[string]interface{} `json:"page_settings,omitempty" swaggertype:"object"`
	UserType              string                 `json:"user_type" example:"organization"`
}

// MobileOrganizationProfileResponse represents /mobile/organization/me.
type MobileOrganizationProfileResponse struct {
	Data MobileOrganizationProfileData `json:"data"`
}

// MobileOrganizationEventItem represents one event owned by the authenticated organization.
type MobileOrganizationEventItem struct {
	UUID               string  `json:"uuid" example:"evt-8f3c2a14-2b73-4a7f-8f7f-2ef1e6c1159a"`
	Name               string  `json:"name" example:"ArcheryHub Jakarta Open 2026"`
	Slug               string  `json:"slug" example:"archeryhub-jakarta-open-2026"`
	Location           *string `json:"location" example:"Jakarta Selatan"`
	Venue              *string `json:"venue" example:"Lapangan ABC Senayan"`
	StartDate          *string `json:"start_date" example:"2026-05-18T08:00:00Z"`
	EndDate            *string `json:"end_date" example:"2026-05-21T17:00:00Z"`
	Status             string  `json:"status" example:"published"`
	LogoURL            *string `json:"logo_url" example:"https://cdn.archeryhub.id/media/events/logo-jkt-open-2026.png"`
	BannerURL          *string `json:"banner_url" example:"https://cdn.archeryhub.id/media/events/banner-jkt-open-2026.jpg"`
	ParticipantCount   int     `json:"participant_count" example:"128"`
	VerifiedCount      int     `json:"verified_count" example:"96"`
	PendingCount       int     `json:"pending_count" example:"32"`
	RegistrationClosed bool    `json:"registration_closed" example:"false"`
}

// MobileOrganizationEventsResponse represents /mobile/organization/events.
type MobileOrganizationEventsResponse struct {
	Events []MobileOrganizationEventItem `json:"events"`
	Total  int                           `json:"total"`
	Limit  int                           `json:"limit"`
	Offset int                           `json:"offset"`
}

// MobileOrganizationParticipantItem represents one participant row with QR info.
type MobileOrganizationParticipantItem struct {
	ID                 string  `json:"id" example:"par-6f0bf699-d807-4ad4-a50d-5d60f7f7ad5d"`
	ArcherID           *string `json:"archer_id" example:"arc-a49ee7d7-9d7b-4be7-8652-342f2fca23f9"`
	AthleteCode        *string `json:"athlete_code" example:"ARC-0042"`
	Username           *string `json:"username" example:"rizky-pratama"`
	FullName           string  `json:"full_name" example:"Rizky Pratama"`
	Email              string  `json:"email" example:"rizky@example.com"`
	City               *string `json:"city" example:"Jakarta"`
	ClubID             *string `json:"club_id" example:"club-1b5d0f48-f3dc-43f3-8ec0-f1fc8805fd29"`
	ClubName           *string `json:"club_name" example:"ArcheryHub Club Jakarta"`
	EventID            string  `json:"event_id" example:"evt-8f3c2a14-2b73-4a7f-8f7f-2ef1e6c1159a"`
	CategoryID         string  `json:"category_id" example:"cat-44f67d53-032f-428f-a8f2-8db5672a7a9d"`
	DivisionName       string  `json:"division_name" example:"Recurve"`
	CategoryName       string  `json:"category_name" example:"Umum"`
	EventTypeName      *string `json:"event_type_name" example:"Individual"`
	GenderDivisionName *string `json:"gender_division_name" example:"Putra"`
	TargetName         *string `json:"target_name" example:"A-12"`
	QRRaw              *string `json:"qr_raw" example:"EVT2026-ARC-0001"`
	QRCodeDataURL      *string `json:"qr_code_data_url" example:"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA..."`
	AvatarURL          *string `json:"avatar_url" example:"https://cdn.archeryhub.id/media/archers/rizky.jpg"`
	RegistrationDate   string  `json:"registration_date" example:"2026-05-01T09:30:00Z"`
	PaymentStatus      string  `json:"payment_status" example:"lunas"`
}

// MobileOrganizationEventParticipantsResponse represents /mobile/organization/events/{id}/participants.
type MobileOrganizationEventParticipantsResponse struct {
	Participants  []MobileOrganizationParticipantItem `json:"participants"`
	Total         int                                 `json:"total"`
	VerifiedCount int                                 `json:"verified_count"`
	PendingCount  int                                 `json:"pending_count"`
	Limit         int                                 `json:"limit"`
	Offset        int                                 `json:"offset"`
}

// MobileUpdateSellerProfileBasicRequest represents basic seller profile update payload.
type MobileUpdateSellerProfileBasicRequest struct {
	StoreName   *string `json:"store_name"`
	Name        *string `json:"name"`
	Username    *string `json:"username"`
	Email       *string `json:"email"`
	Phone       *string `json:"phone"`
	City        *string `json:"city"`
	Province    *string `json:"province"`
	Address     *string `json:"address"`
	Description *string `json:"description"`
	AvatarURL   *string `json:"avatar_url"`
	BannerURL   *string `json:"banner_url"`
	Logo        *string `json:"logo"`
	Banner      *string `json:"banner"`
}

// MobileUpdateSellerPageRequest represents seller page settings update payload.
type MobileUpdateSellerPageRequest struct {
	Sections      interface{} `json:"sections"`
	CatalogConfig interface{} `json:"catalog_config"`
	ThemeColor    string      `json:"theme_color"`
	BannerText    string      `json:"banner_text"`
	PageSettings  interface{} `json:"page_settings"`
}

// MobileUpdateOrganizationProfileRequest represents organization profile update payload.
type MobileUpdateOrganizationProfileRequest struct {
	Slug               *string     `json:"slug"`
	Name               *string     `json:"name"`
	Acronym            *string     `json:"acronym"`
	Description        *string     `json:"description"`
	Website            *string     `json:"website"`
	WhatsAppNo         *string     `json:"whatsapp_no"`
	AvatarURL          *string     `json:"avatar_url"`
	BannerURL          *string     `json:"banner_url"`
	Address            *string     `json:"address"`
	City               *string     `json:"city"`
	Country            *string     `json:"country"`
	RegistrationNumber *string     `json:"registration_number"`
	EstablishedDate    *string     `json:"established_date"`
	ContactPersonName  *string     `json:"contact_person_name"`
	ContactPersonEmail *string     `json:"contact_person_email"`
	ContactPersonPhone *string     `json:"contact_person_phone"`
	SocialFacebook     *string     `json:"social_facebook"`
	SocialInstagram    *string     `json:"social_instagram"`
	SocialTwitter      *string     `json:"social_twitter"`
	SocialMedia        interface{} `json:"social_media"`
	Vision             *string     `json:"vision"`
	Mission            *string     `json:"mission"`
	History            *string     `json:"history"`
	FAQ                interface{} `json:"faq"`
	PageSettings       interface{} `json:"page_settings"`
}

// MobileArcherProfileData represents archer profile data for mobile.
type MobileArcherProfileData struct {
	ID          string  `json:"id" example:"ARC-0042"`
	UUID        string  `json:"uuid" example:"arc-a49ee7d7-9d7b-4be7-8652-342f2fca23f9"`
	Username    *string `json:"username" example:"rizky-pratama"`
	FullName    string  `json:"full_name" example:"Rizky Pratama"`
	Email       *string `json:"email" example:"rizky@example.com"`
	AvatarURL   *string `json:"avatar_url" example:"https://cdn.archeryhub.id/media/archers/rizky.jpg"`
	Phone       *string `json:"phone" example:"081234567890"`
	Gender      *string `json:"gender" example:"male"`
	DateOfBirth *string `json:"date_of_birth" example:"1995-05-18"`
	City        *string `json:"city" example:"Jakarta"`
	Address     *string `json:"address" example:"Jl. Panahan No. 10"`
	BowType     *string `json:"bow_type" example:"recurve"`
	ClubID      *string `json:"club_id" example:"club-1b5d0f48-f3dc-43f3-8ec0-f1fc8805fd29"`
	ClubName    *string `json:"club_name" example:"ArcheryHub Club Jakarta"`
	UserType    string  `json:"user_type" example:"archer"`
}

// MobileArcherProfileResponse represents /mobile/archer/me response.
type MobileArcherProfileResponse struct {
	Data MobileArcherProfileData `json:"data"`
}

// MobileUpdateArcherProfileRequest represents archer profile update payload.
type MobileUpdateArcherProfileRequest struct {
	FullName    *string `json:"full_name"`
	Phone       *string `json:"phone"`
	Gender      *string `json:"gender"`
	DateOfBirth *string `json:"date_of_birth"`
	City        *string `json:"city"`
	Address     *string `json:"address"`
	BowType     *string `json:"bow_type"`
	AvatarURL   *string `json:"avatar_url"`
}

// MobileOrganizationScanRegistrationRequest represents registration scan payload.
type MobileOrganizationScanRegistrationRequest struct {
	Code string `json:"code" binding:"required" example:"REG-f93c2a14-2b73-4a7f-8f7f-2ef1e6c1159a"`
}

// MobileOrganizationScanRegistrationResponse represents scan result.
type MobileOrganizationScanRegistrationResponse struct {
	ParticipantUUID      string  `json:"participant_uuid" example:"f93c2a14-2b73-4a7f-8f7f-2ef1e6c1159a"`
	FullName             string  `json:"full_name" example:"Rizky Pratama"`
	AthleteCode          string  `json:"athlete_code" example:"ARC-0042"`
	EventName            string  `json:"event_name" example:"ArcheryHub Jakarta Open 2026"`
	CategoryName         string  `json:"category_name" example:"Recurve Umum Putra"`
	ClubName             *string `json:"club_name" example:"ArcheryHub Club Jakarta"`
	PaymentStatus        string  `json:"payment_status" example:"lunas"`
	LastReregistrationAt *string `json:"last_reregistration_at" example:"2026-03-27T10:00:00Z"`
}
