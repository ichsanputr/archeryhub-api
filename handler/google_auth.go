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

		// Generate state for CSRF protection
		stateBytes := make([]byte, 16)
		if _, err := rand.Read(stateBytes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate state"})
			return
		}
		state := hex.EncodeToString(stateBytes)

		// Store state with app URL for callback
		// In production, use Redis or database
		stateData := fmt.Sprintf("%s|%s", state, appURL)

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
		code := c.Query("code")
		stateData := c.Query("state")

		if code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No authorization code provided"})
			return
		}

		// Parse state to get app URL
		appURL := ""
		if len(stateData) > 33 { // state is 32 chars + "|"
			appURL = stateData[33:]
		}
		if appURL == "" {
			appURL = os.Getenv("APP_URL")
		}

		// Exchange code for token
		tokenResponse, err := exchangeGoogleCode(code)
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, appURL+"/auth/login?error=token_exchange_failed")
			return
		}

		// Get user info from Google
		userInfo, err := getGoogleUserInfo(tokenResponse.AccessToken)
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, appURL+"/auth/login?error=user_info_failed")
			return
		}

		// Find or create user
		var userID string
		var userExists bool

		err = db.Get(&userExists, "SELECT EXISTS(SELECT 1 FROM users WHERE email = ? OR google_id = ?)", userInfo.Email, userInfo.ID)
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, appURL+"/auth/login?error=database_error")
			return
		}

		if userExists {
			// Update existing user
			err = db.Get(&userID, "SELECT id FROM users WHERE email = ? OR google_id = ?", userInfo.Email, userInfo.ID)
			if err != nil {
				c.Redirect(http.StatusTemporaryRedirect, appURL+"/auth/login?error=user_lookup_failed")
				return
			}

			// Update Google-specific fields
			_, err = db.Exec(`
				UPDATE users 
				SET google_id = ?, avatar_url = ?, full_name = ?, updated_at = NOW()
				WHERE id = ?
			`, userInfo.ID, userInfo.Picture, userInfo.Name, userID)
			if err != nil {
				// Log but don't fail
				fmt.Printf("Failed to update user: %v\n", err)
			}
		} else {
			// Create new user
			userID = uuid.New().String()
			username := generateUsername(userInfo.Email)

			_, err = db.Exec(`
				INSERT INTO users (id, username, email, google_id, full_name, avatar_url, role, status, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, 'athlete', 'active', NOW(), NOW())
			`, userID, username, userInfo.Email, userInfo.ID, userInfo.Name, userInfo.Picture)

			if err != nil {
				c.Redirect(http.StatusTemporaryRedirect, appURL+"/auth/login?error=user_creation_failed")
				return
			}

			// Log activity
			utils.LogActivity(db, userID, "", "user_registered", "user", userID, "User registered via Google: "+userInfo.Email, c.ClientIP(), c.Request.UserAgent())
		}

		// Generate JWT token
		token, err := generateGoogleJWT(userID, userInfo.Email, "athlete")
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, appURL+"/auth/login?error=token_generation_failed")
			return
		}

		// Log activity
		utils.LogActivity(db, userID, "", "user_logged_in", "user", userID, "User logged in via Google", c.ClientIP(), c.Request.UserAgent())

		// Set cookie and redirect
		isProduction := os.Getenv("ENV") == "production"
		domain := ""
		if isProduction {
			domain = ".archeryhub.id"
		}

		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("auth_token", token, 60*60*24*7, "/", domain, isProduction, true)

		// Redirect to app with token
		c.Redirect(http.StatusTemporaryRedirect, appURL+"?token="+token)
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

// generateGoogleJWT generates a JWT token for Google OAuth users
func generateGoogleJWT(userID, email, role string) (string, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) == 0 {
		secret = []byte("archeryhub-secret-key-change-in-production")
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    role,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}
