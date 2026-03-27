package mobile

import "archeryhub-api/models"

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
