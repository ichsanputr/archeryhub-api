package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"archeryhub-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// GoogleOAuthConfig holds Google OAuth configuration
type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// GoogleTokenResponse represents the response from Google's token endpoint
type GoogleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

// GoogleUserInfo represents user info from Google
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// InitiateGoogleAuth initiates the Google OAuth flow
func InitiateGoogleAuth(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := os.Getenv("GOOGLE_CLIENT_ID")
		if clientID == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Google OAuth not configured"})
			return
		}

		appURL := c.Query("app_url")
		if appURL == "" {
			appURL = os.Getenv("APP_URL")
		}

		// Get user type for registration (defaults to archer)
		userType := c.Query("user_type")
		if userType == "" {
			userType = "archer"
		}
		// Validate user type
		if userType != "archer" && userType != "organization" && userType != "club" && userType != "seller" {
			userType = "archer"
		}

		// Get full name if provided
		fullName := c.Query("full_name")

		// Generate state for CSRF protection
		stateBytes := make([]byte, 16)
		if _, err := rand.Read(stateBytes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate state"})
			return
		}
		state := hex.EncodeToString(stateBytes)

		// Store state with app URL, user_type, and fullName for callback
		// Format: state|appURL|userType|fullName
		stateData := fmt.Sprintf("%s|%s|%s|%s", state, appURL, userType, fullName)

		redirectURI := os.Getenv("GOOGLE_REDIRECT_URI")
		if redirectURI == "" {
			redirectURI = os.Getenv("API_URL") + "/auth/google/callback"
		}

		// Build Google OAuth URL
		authURL := fmt.Sprintf(
			"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s&access_type=offline&prompt=consent",
			url.QueryEscape(clientID),
			url.QueryEscape(redirectURI),
			url.QueryEscape("openid email profile"),
			url.QueryEscape(stateData),
		)

		c.JSON(http.StatusOK, gin.H{
			"auth_url": authURL,
			"state":    state,
		})
	}
}

// GoogleCallback handles the OAuth callback from Google
func GoogleCallback(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var code string
		var stateData string

		// Try to get from JSON first (POST)
		var body struct {
			Code  string `json:"code"`
			State string `json:"state"`
		}
		if err := c.ShouldBindJSON(&body); err == nil && body.Code != "" {
			code = body.Code
			stateData = body.State
		} else {
			// Fallback to query params (GET)
			code = c.Query("code")
			stateData = c.Query("state")
		}

		if code == "" {
			if c.ContentType() == "application/json" || c.GetHeader("Accept") == "application/json" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "No authorization code provided"})
			} else {
				c.Redirect(http.StatusTemporaryRedirect, os.Getenv("APP_URL")+"/auth/login?error=no_code")
			}
			return
		}

		// Parse state to get app URL, user_type, and fullName
		// Format: state|appURL|userType|fullName
		appURL := ""
		requestedUserType := "archer" // default
		requestedFullName := ""
		parts := splitState(stateData)
		if len(parts) >= 2 {
			appURL = parts[1]
		}
		if len(parts) >= 3 {
			requestedUserType = parts[2]
		}
		if len(parts) >= 4 {
			requestedFullName = parts[3]
		}
		if appURL == "" {
			appURL = os.Getenv("APP_URL")
		}

		// Exchange code for token
		tokenResponse, err := exchangeGoogleCode(code)
		if err != nil {
			msg := "token_exchange_failed"
			if c.ContentType() == "application/json" || c.GetHeader("Accept") == "application/json" {
				c.JSON(http.StatusInternalServerError, gin.H{"error": msg, "details": err.Error()})
			} else {
				c.Redirect(http.StatusTemporaryRedirect, appURL+"/auth/login?error="+msg)
			}
			return
		}

		// Get user info from Google
		userInfo, err := getGoogleUserInfo(tokenResponse.AccessToken)
		if err != nil {
			msg := "user_info_failed"
			if c.ContentType() == "application/json" || c.GetHeader("Accept") == "application/json" {
				c.JSON(http.StatusInternalServerError, gin.H{"error": msg, "details": err.Error()})
			} else {
				c.Redirect(http.StatusTemporaryRedirect, appURL+"/auth/login?error="+msg)
			}
			return
		}

		// Find or create user
		var userID string
		var userType string
		var role string
		found := false
		isNewUser := false

		// Find existing user across all tables
		type UserRecord struct {
			UUID string `db:"uuid"`
			Type string
			Role string `db:"role"`
		}
		var record UserRecord

		// Priority search
		tables := []string{"archers", "organizations", "clubs", "sellers"}
		for _, t := range tables {
			typeToRole := t
			if typeToRole == "archers" {
				typeToRole = "archer"
			}
			if typeToRole == "organizations" {
				typeToRole = "organization"
			}
			if typeToRole == "clubs" {
				typeToRole = "club"
			}
			if typeToRole == "sellers" {
				typeToRole = "seller"
			}

			query := "SELECT uuid, '" + typeToRole + "' as role FROM " + t + " WHERE email = ? OR google_id = ?"
			err = db.Get(&record, query, userInfo.Email, userInfo.ID)
			if err == nil && record.UUID != "" {
				userType = typeToRole
				userID = record.UUID
				role = record.Role
				found = true
				break
			}
		}

		if found {
			// Register flow with existing email: user came from register page (full_name in state) but email already exists
			if requestedFullName != "" {
				if c.ContentType() == "application/json" || c.GetHeader("Accept") == "application/json" || c.Request.Method == "POST" {
					c.JSON(http.StatusConflict, gin.H{
						"already_registered": true,
						"email":              userInfo.Email,
						"user_type":          userType,
					})
				} else {
					redirectURL := fmt.Sprintf("%s/auth/already-registered?email=%s&user_type=%s", appURL, url.QueryEscape(userInfo.Email), url.QueryEscape(userType))
					c.Redirect(http.StatusTemporaryRedirect, redirectURL)
				}
				return
			}

			// Login flow: update only Google-specific fields; do NOT overwrite existing name
			table := ""
			switch userType {
			case "organization":
				table = "organizations"
			case "club":
				table = "clubs"
			case "seller":
				table = "sellers"
			default:
				table = "archers"
			}
			_, err = db.Exec(`
				UPDATE `+table+` 
				SET google_id = ?, avatar_url = ?, updated_at = NOW()
				WHERE uuid = ?
			`, userInfo.ID, userInfo.Picture, userID)
			if err != nil {
				fmt.Printf("Failed to update user in %s: %v\n", table, err)
			}
		}

		// Resolve display name for JWT: existing user = from DB; new user = requestedFullName or Google name
		displayNameForJWT := userInfo.Name
		if found {
			var nameCol, tableName string
			switch userType {
			case "organization":
				tableName, nameCol = "organizations", "name"
			case "club":
				tableName, nameCol = "clubs", "name"
			case "seller":
				tableName, nameCol = "sellers", "store_name"
			default:
				tableName, nameCol = "archers", "full_name"
			}
			var existingName string
			if qErr := db.Get(&existingName, "SELECT "+nameCol+" FROM "+tableName+" WHERE uuid = ?", userID); qErr == nil && existingName != "" {
				displayNameForJWT = existingName
			}
		}

		if !found {
			// Create new user based on requested user type
			userID = uuid.New().String()
			userType = requestedUserType
			role = requestedUserType
			isNewUser = true
			username := generateUsername(userInfo.Email)

			// Use requestedFullName if provided, otherwise use Google name
			displayName := userInfo.Name
			if requestedFullName != "" {
				displayName = requestedFullName
			}
			displayNameForJWT = displayName

			var insertErr error
			switch userType {
			case "organization":
				_, insertErr = db.Exec(`
					INSERT INTO organizations (uuid, slug, email, google_id, name, avatar_url, status, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, ?, 'active', NOW(), NOW())
				`, userID, username, userInfo.Email, userInfo.ID, displayName, userInfo.Picture)
			case "club":
				_, insertErr = db.Exec(`
					INSERT INTO clubs (uuid, slug, email, google_id, name, avatar_url, status, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, ?, 'active', NOW(), NOW())
				`, userID, username, userInfo.Email, userInfo.ID, displayName, userInfo.Picture)
			case "seller":
				_, insertErr = db.Exec(`
					INSERT INTO sellers (uuid, slug, email, google_id, store_name, avatar_url, status, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, ?, 'active', NOW(), NOW())
				`, userID, username, userInfo.Email, userInfo.ID, displayName, userInfo.Picture)
			default: // archer
				userType = "archer"
				role = "archer"
				// Generate slug from name
				slug := strings.ToLower(displayName)
				slug = strings.ReplaceAll(slug, " ", "-")
				slug = slug + "-" + userID[:8]

				_, insertErr = db.Exec(`
					INSERT INTO archers (uuid, slug, email, google_id, full_name, avatar_url, status, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, ?, 'active', NOW(), NOW())
				`, userID, slug, userInfo.Email, userInfo.ID, displayName, userInfo.Picture)
			}

			if insertErr != nil {
				if c.ContentType() == "application/json" || c.GetHeader("Accept") == "application/json" {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "user_creation_failed", "details": insertErr.Error()})
				} else {
					c.Redirect(http.StatusTemporaryRedirect, appURL+"/auth/login?error=user_creation_failed")
				}
				return
			}

			// Log activity
			utils.LogActivity(db, userID, "", "user_registered", userType, userID, "User registered via Google: "+userInfo.Email, c.ClientIP(), c.Request.UserAgent())
		}

		// Generate JWT token (use displayNameForJWT so existing user keeps their name)
		token, err := generateGoogleJWT(userID, userInfo.Email, role, userType, displayNameForJWT, userInfo.Picture)
		if err != nil {
			if c.ContentType() == "application/json" || c.GetHeader("Accept") == "application/json" {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "token_generation_failed"})
			} else {
				c.Redirect(http.StatusTemporaryRedirect, appURL+"/auth/login?error=token_generation_failed")
			}
			return
		}

		// Log activity
		utils.LogActivity(db, userID, "", "user_logged_in", userType, userID, "User logged in via Google", c.ClientIP(), c.Request.UserAgent())

		// Set cookie
		isProduction := os.Getenv("ENV") == "production"
		domain := ""
		if isProduction {
			domain = ".archeryhub.id"
		}

		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("auth_token", token, 60*60*24*7, "/", domain, isProduction, true)

		// Return response based on request type
		if c.ContentType() == "application/json" || c.GetHeader("Accept") == "application/json" || c.Request.Method == "POST" {
			c.JSON(http.StatusOK, gin.H{
				"token":       token,
				"is_new_user": isNewUser,
				"user": gin.H{
					"id":         userID,
					"email":      userInfo.Email,
					"full_name":  displayNameForJWT,
					"avatar_url": userInfo.Picture,
					"role":       role,
					"user_type":  userType,
				},
			})

		} else {
			// Redirect back to app
			target := appURL + "?token=" + token
			c.Redirect(http.StatusTemporaryRedirect, target)
		}
	}
}

// exchangeGoogleCode exchanges authorization code for tokens
func exchangeGoogleCode(code string) (*GoogleTokenResponse, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURI := os.Getenv("GOOGLE_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = os.Getenv("API_URL") + "/auth/google/callback"
	}

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", redirectURI)

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp GoogleTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// getGoogleUserInfo retrieves user info from Google
func getGoogleUserInfo(accessToken string) (*GoogleUserInfo, error) {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: %s", string(body))
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// generateUsername creates a username from email
func generateUsername(email string) string {
	// Extract part before @
	for i, c := range email {
		if c == '@' {
			return email[:i]
		}
	}
	return email
}

// splitState splits the OAuth state string into parts
func splitState(stateData string) []string {
	result := []string{}
	current := ""
	for _, c := range stateData {
		if c == '|' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// generateGoogleJWT generates a JWT token for Google OAuth users
func generateGoogleJWT(userID, email, role, userType, name, avatar string) (string, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) == 0 {
		secret = []byte("archeryhub-secret-key-change-in-production")
	}

	claims := jwt.MapClaims{
		"user_id":   userID,
		"email":     email,
		"name":      name,
		"avatar":    avatar,
		"role":      role,
		"user_type": userType,
		"exp":       time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
		"iat":       time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}
