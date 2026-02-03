package handler

import (
	"archeryhub-api/models"
	"archeryhub-api/utils"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// GetArchers returns a list of archers with optional filtering
func GetArchers(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Query("status")
		search := c.Query("search") // search by name, code, or club
		city := c.Query("city")
		bowType := c.Query("bow_type")
		limit := c.DefaultQuery("limit", "12")
		offset := c.DefaultQuery("offset", "0")

		limitInt, _ := strconv.Atoi(limit)
		offsetInt, _ := strconv.Atoi(offset)

		query := `
			SELECT 
				a.uuid, a.id, a.username, a.full_name, a.date_of_birth,
				a.gender, a.email, a.phone, a.avatar_url, a.address,
				a.bio, a.status, a.created_at, a.updated_at,
				a.bow_type, a.city, a.school, a.province,
				c.name as club_name,
				c.slug as club_slug,
				COUNT(DISTINCT tp.uuid) as total_events,
				COUNT(DISTINCT CASE WHEN t.status = 'completed' THEN tp.uuid END) as completed_events,
				MAX(t.end_date) as last_event_date
			FROM archers a
			LEFT JOIN clubs c ON a.club_id = c.uuid
			LEFT JOIN event_participants tp ON a.uuid = tp.archer_id
			LEFT JOIN events t ON tp.event_id = t.uuid
			WHERE 1=1
		`
		args := []interface{}{}

		if status != "" {
			query += " AND a.status = ?"
			args = append(args, status)
		}

		if search != "" {
			query += " AND (a.full_name LIKE ? OR a.email LIKE ? OR a.club_id LIKE ?)"
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm, searchTerm)
		}

		if city != "" {
			query += " AND a.city = ?"
			args = append(args, city)
		}

		if bowType != "" && bowType != "all" {
			query += " AND a.bow_type = ?"
			args = append(args, bowType)
		}

		query += `
			GROUP BY a.uuid
			ORDER BY a.full_name
			LIMIT ? OFFSET ?
		`
		args = append(args, limitInt, offsetInt)

		var archers []models.ArcherWithStats
		err := db.Select(&archers, query, args...)

		if err != nil {
			logrus.WithError(err).Error("Failed to fetch archers")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch archers", "details": err.Error()})
			return
		}

		// Get total count
		countQuery := `SELECT COUNT(*) FROM archers WHERE 1=1`
		countArgs := []interface{}{}

		if status != "" {
			countQuery += " AND status = ?"
			countArgs = append(countArgs, status)
		}

		if search != "" {
			countQuery += " AND (full_name LIKE ? OR email LIKE ? OR club_id LIKE ?)"
			searchTerm := "%" + search + "%"
			countArgs = append(countArgs, searchTerm, searchTerm, searchTerm)
		}

		if city != "" {
			countQuery += " AND city = ?"
			countArgs = append(countArgs, city)
		}

		if bowType != "" && bowType != "all" {
			countQuery += " AND bow_type = ?"
			countArgs = append(countArgs, bowType)
		}

		var total int
		err = db.Get(&total, countQuery, countArgs...)
		if err != nil {
			logrus.WithError(err).Error("Failed to count archers")
		}

		// Mask URLs
		for i := range archers {
			if archers[i].AvatarURL != nil {
				masked := utils.MaskMediaURL(*archers[i].AvatarURL)
				archers[i].AvatarURL = &masked
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"archers":  archers,
			"athletes": archers,
			"count":    len(archers),
			"total":    total,
		})
	}
}

// GetArcherByID returns a single archer by ID or slug
func GetArcherByID(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		query := `
			SELECT 
				a.uuid, a.id, a.username, a.full_name, a.date_of_birth,
				a.gender, a.email, a.phone, a.avatar_url, a.address,
				a.bio, a.status, a.created_at, a.updated_at,
				a.bow_type, a.city, a.school, a.province,
				c.name as club_name,
				c.slug as club_slug,
				COUNT(DISTINCT tp.uuid) as total_events,
				COUNT(DISTINCT CASE WHEN t.status = 'completed' THEN tp.uuid END) as completed_events,
				MAX(t.end_date) as last_event_date
			FROM archers a
			LEFT JOIN clubs c ON a.club_id = c.uuid
			LEFT JOIN event_participants tp ON a.uuid = tp.archer_id
			LEFT JOIN events t ON tp.event_id = t.uuid
			WHERE a.uuid = ? OR a.username = ? OR (a.id != '' AND a.id = ?)
			GROUP BY a.uuid
			LIMIT 1
		`

		var archer models.ArcherWithStats
		err := db.Get(&archer, query, id, id, id)
		if err != nil {
			logrus.WithError(err).Warnf("Archer not found: %s", id)
			c.JSON(http.StatusNotFound, gin.H{"error": "Archer not found"})
			return
		}

		// Mask URLs
		if archer.AvatarURL != nil {
			masked := utils.MaskMediaURL(*archer.AvatarURL)
			archer.AvatarURL = &masked
		}

		c.JSON(http.StatusOK, archer)
	}
}

type ArcherEventHistory struct {
	ID        string     `json:"id" db:"id"`
	Name      string     `json:"name" db:"name"`
	City      *string    `json:"city" db:"city"`
	StartDate *time.Time `json:"date" db:"start_date"`
	Score     *int       `json:"score" db:"qual_score"`
	Rank      *int       `json:"rank" db:"qual_rank"`
}

// GetArcherEvents returns the event history for a specific archer
func GetArcherEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		query := `
			SELECT 
				e.uuid as id, e.name as name, e.city, e.start_date as start_date, 
				0 as qual_score, 0 as qual_rank
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			JOIN archers a ON ep.archer_id = a.uuid
			WHERE a.uuid = ? OR a.username = ? OR (a.id != '' AND a.id = ?)
			ORDER BY e.start_date DESC
		`

		var events []ArcherEventHistory
		err := db.Select(&events, query, id, id, id)
		if err != nil {
			logrus.WithError(err).Error("Failed to fetch archer events")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch archer events", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"events": events,
		})
	}
}

// GetMyArcherEvents returns events that the authenticated archer is registered for
func GetMyArcherEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		status := c.Query("status")
		search := c.Query("search")

		query := `
			SELECT 
				e.*,
				o.full_name as organizer_name,
				o.email as organizer_email,
				o.slug as organizer_slug,
				o.avatar_url as organizer_avatar_url,
				COUNT(DISTINCT ep2.uuid) as participant_count,
				COUNT(DISTINCT ec.uuid) as event_count,
				ep.payment_status,
				ep.uuid as participant_uuid,
				ep.status as participant_status,
				ep.qr_raw
			FROM events e
			INNER JOIN event_participants ep ON e.uuid = ep.event_id
			LEFT JOIN (
				SELECT uuid as id, name as full_name, email, slug, avatar_url FROM organizations
				UNION ALL
				SELECT uuid as id, name as full_name, email, slug, avatar_url FROM clubs
			) o ON e.organizer_id = o.id
			LEFT JOIN event_participants ep2 ON e.uuid = ep2.event_id
			LEFT JOIN event_categories ec ON e.uuid = ec.event_id
			WHERE ep.archer_id = ?
		`
		args := []interface{}{userID}

		if status != "" {
			query += ` AND e.status = ?`
			args = append(args, status)
		}

		if search != "" {
			query += ` AND (e.name LIKE ? OR e.code LIKE ? OR e.location LIKE ?)`
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm, searchTerm)
		}

		query += `
			GROUP BY e.uuid, ep.payment_status, ep.uuid, ep.status, ep.qr_raw, o.full_name, o.email, o.slug, o.avatar_url
			ORDER BY e.start_date DESC
		`

		var events []models.EventWithDetails
		err := db.Select(&events, query, args...)
		if err != nil {
			logrus.WithError(err).Error("Failed to fetch archer events")
			c.JSON(http.StatusOK, gin.H{
				"events": []interface{}{},
				"total":  0,
			})
			return
		}

		// Mask URLs
		for i := range events {
			if events[i].BannerURL != nil {
				masked := utils.MaskMediaURL(*events[i].BannerURL)
				events[i].BannerURL = &masked
			}
			if events[i].LogoURL != nil {
				masked := utils.MaskMediaURL(*events[i].LogoURL)
				events[i].LogoURL = &masked
			}
			if events[i].TechnicalGuidebookURL != nil {
				masked := utils.MaskMediaURL(*events[i].TechnicalGuidebookURL)
				events[i].TechnicalGuidebookURL = &masked
			}
			if events[i].OrganizerAvatarURL != nil {
				masked := utils.MaskMediaURL(*events[i].OrganizerAvatarURL)
				events[i].OrganizerAvatarURL = &masked
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"events": events,
			"total":  len(events),
		})
	}
}

// CreateArcher creates a new archer
func CreateArcher(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.CreateArcherRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		archerID := uuid.New().String()
		now := time.Now()

		// Check if email/username already exists
		if req.Email != nil {
			var exists bool
			err := db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM archers WHERE email = ?)", req.Email)
			if err == nil && exists {
				c.JSON(http.StatusConflict, gin.H{"error": "Email or username already exists"})
				return
			}
		}

		// Validate password length if provided
		if req.Password != nil && *req.Password != "" {
			if len(*req.Password) < 6 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Password harus minimal 6 karakter"})
				return
			}
		}

		// Generate archer code removed as athlete_code column is deleted

		// Normalize gender (M/F to male/female)
		gender := req.Gender
		if gender != nil {
			if *gender == "M" {
				g := "male"
				gender = &g
			} else if *gender == "F" {
				g := "female"
				gender = &g
			}
		}

		// Get club_id from request or from logged-in user if user is a club
		var clubID *string
		userID, _ := c.Get("user_id")
		if req.ClubID != nil {
			clubID = req.ClubID
		} else {
			userType, _ := c.Get("user_type")
			if userType == "club" && userID != nil {
				clubIDStr := userID.(string)
				clubID = &clubIDStr
			}
		}

		// Generate id (ARC-XXXX)
		var lastID string
		_ = db.Get(&lastID, "SELECT id FROM archers WHERE id LIKE 'ARC-%' ORDER BY id DESC LIMIT 1")
		nextIDNum := 1
		if lastID != "" {
			parts := strings.Split(lastID, "-")
			if len(parts) == 2 {
				fmt.Sscanf(parts[1], "%d", &nextIDNum)
				nextIDNum++
			}
		}
		athleteID := fmt.Sprintf("ARC-%04d", nextIDNum)
		if req.ID != nil && *req.ID != "" {
			athleteID = *req.ID
		}

		// Generate username if not provided
		var username string
		if req.Username != nil && *req.Username != "" {
			username = *req.Username
		} else {
			// Generate username from full name
			generatedUsername := strings.ToLower(req.FullName)
			generatedUsername = strings.ReplaceAll(generatedUsername, " ", "-")
			generatedUsername = strings.ReplaceAll(generatedUsername, "'", "")
			generatedUsername = strings.ReplaceAll(generatedUsername, ".", "")
			generatedUsername = strings.ReplaceAll(generatedUsername, ",", "")
			// Remove special characters, keep only alphanumeric and hyphens
			var cleaned strings.Builder
			for _, r := range generatedUsername {
				if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
					cleaned.WriteRune(r)
				}
			}
			generatedUsername = cleaned.String()
			if generatedUsername == "" {
				generatedUsername = "archer"
			}
			username = generatedUsername
		}

		// Check if username already exists, if so add a suffix
		var finalUsername string = username
		var usernameExists bool
		_ = db.Get(&usernameExists, "SELECT EXISTS(SELECT 1 FROM archers WHERE username = ?)", finalUsername)

		if usernameExists {
			// Add random suffix for uniqueness only if conflict
			randomSuffix := uuid.New().String()[:8]
			finalUsername = fmt.Sprintf("%s-%s", username, randomSuffix)
		}

		// Set verification status: Unverified if no password
		isVerified := false
		if req.Password != nil && *req.Password != "" {
			isVerified = true
		}

		query := `
			INSERT INTO archers (
				uuid, id, username, email, password, full_name, nickname,
				date_of_birth, gender, bow_type, city, school, club_id,
				phone, address, avatar_url, status, is_verified, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?, ?, ?)
		`

		_, err := db.Exec(query,
			archerID, athleteID, finalUsername, req.Email, req.Password, req.FullName, req.Nickname,
			req.DateOfBirth, gender, req.BowType, req.City, req.School, clubID,
			req.Phone, req.Address, req.AvatarURL, isVerified, now, now,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create archer", "details": err.Error()})
			return
		}

		// If created by a club, create club_members entry
		if clubID != nil {
			memberID := uuid.New().String()
			_, err = db.Exec(`
				INSERT INTO club_members (uuid, club_id, archer_id, status, role, created_at)
				VALUES (?, ?, ?, 'active', 'member', NOW())
			`, memberID, *clubID, archerID)
			if err != nil {
				// Log error but don't fail the request
				utils.LogActivity(db, userID.(string), "", "club_member_link_failed", "archer", archerID, "Failed to link archer to club: "+err.Error(), c.ClientIP(), c.Request.UserAgent())
			}
		}

		// Log activity
		if userID != nil {
			utils.LogActivity(db, userID.(string), "", "archer_created", "archer", archerID, "Created new archer: "+req.FullName, c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":   "Archer created successfully",
			"archer_id": archerID,
		})
	}
}

// UpdateArcher updates an existing archer
func UpdateArcher(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req models.UpdateArcherRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Check if archer exists
		var exists bool
		err := db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM archers WHERE uuid = ?)", id)
		if err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Archer not found"})
			return
		}

		// Build dynamic update query
		query := "UPDATE archers SET updated_at = NOW()"
		args := []interface{}{}

		if req.FullName != nil {
			query += ", full_name = ?"
			args = append(args, *req.FullName)
		}
		if req.DateOfBirth != nil {
			query += ", date_of_birth = ?"
			args = append(args, *req.DateOfBirth)
		}
		if req.Gender != nil {
			query += ", gender = ?"
			args = append(args, *req.Gender)
		}
		if req.City != nil {
			query += ", city = ?"
			args = append(args, *req.City)
		}
		if req.BowType != nil {
			query += ", bow_type = ?"
			args = append(args, *req.BowType)
		}
		if req.School != nil {
			query += ", school = ?"
			args = append(args, *req.School)
		}
		if req.ClubID != nil {
			query += ", club_id = ?"
			args = append(args, *req.ClubID)
		}
		if req.Email != nil {
			query += ", email = ?"
			args = append(args, *req.Email)
		}
		if req.Phone != nil {
			query += ", phone = ?"
			args = append(args, *req.Phone)
		}
		if req.AvatarURL != nil {
			query += ", avatar_url = ?"
			args = append(args, *req.AvatarURL)
		}
		if req.Status != nil {
			query += ", status = ?"
			args = append(args, *req.Status)
		}

		query += " WHERE uuid = ?"
		args = append(args, id)

		_, err = db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update archer", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		if userID != nil {
			utils.LogActivity(db, userID.(string), "", "archer_updated", "archer", id, "Updated archer", c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{"message": "Archer updated successfully"})
	}
}

// DeleteArcher deletes an archer
func DeleteArcher(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// Check if archer has any event participations
		var participationCount int
		db.Get(&participationCount, "SELECT COUNT(*) FROM event_participants WHERE archer_id = ?", id)

		if participationCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete archer with event participations"})
			return
		}

		result, err := db.Exec("DELETE FROM archers WHERE uuid = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete archer", "details": err.Error()})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Archer not found"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		if userID != nil {
			utils.LogActivity(db, userID.(string), "", "archer_deleted", "archer", id, "Deleted archer", c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{"message": "Archer deleted successfully"})
	}
}

// GetArcherProfile returns the authenticated archer's profile
func GetArcherProfile(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var archer struct {
			UUID        string  `json:"uuid" db:"uuid"`
			ID          string  `json:"id" db:"id"`
			Username    *string `json:"username" db:"username"`
			Email       *string `json:"email" db:"email"`
			AvatarURL   *string `json:"avatar_url" db:"avatar_url"`
			FullName    string  `json:"full_name" db:"full_name"`
			Nickname    *string `json:"nickname" db:"nickname"`
			DateOfBirth *string `json:"date_of_birth" db:"date_of_birth"`
			Gender      string  `json:"gender" db:"gender"`
			Phone       *string `json:"phone" db:"phone"`
			Address     *string `json:"address" db:"address"`
			City        *string `json:"city" db:"city"`
			School      *string `json:"school" db:"school"`
			Province    *string `json:"province" db:"province"`
			BowType     string  `json:"bow_type" db:"bow_type"`
			ClubID      *string `json:"club_id" db:"club_id"`
			ClubName    *string `json:"club_name" db:"club_name"`
			Status      string  `json:"status" db:"status"`
		}

		var pageSettings *string
		err := db.Get(&archer, `
		SELECT a.uuid, a.id, a.username, a.email, a.avatar_url, 
		       a.full_name, a.nickname, a.date_of_birth, 
		       COALESCE(a.gender, 'male') as gender,
		       a.phone, a.address, a.city, a.school, a.province, 
		       COALESCE(a.bow_type, 'recurve') as bow_type,
		       a.club_id, c.name as club_name,
		       COALESCE(a.status, 'active') as status
		FROM archers a
		LEFT JOIN clubs c ON a.club_id = c.uuid
		WHERE a.uuid = ? OR a.email = (SELECT email FROM archers WHERE uuid = ? LIMIT 1)
	`, userID, userID)

		if err == nil {
			db.Get(&pageSettings, "SELECT page_settings FROM archers WHERE uuid = ?", userID)
		}

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Archer profile not found"})
			return
		}

		data := gin.H{
			"uuid":          archer.UUID,
			"id":            archer.ID,
			"username":      archer.Username,
			"email":         archer.Email,
			"avatar_url":    archer.AvatarURL,
			"full_name":     archer.FullName,
			"nickname":      archer.Nickname,
			"date_of_birth": archer.DateOfBirth,
			"gender":        archer.Gender,
			"phone":         archer.Phone,
			"address":       archer.Address,
			"city":          archer.City,
			"school":        archer.School,
			"province":      archer.Province,
			"bow_type":      archer.BowType,
			"club_id":       archer.ClubID,
			"club_name":     archer.ClubName,
			"status":        archer.Status,
			"user_type":     "archer",
		}

		if pageSettings != nil {
			data["page_settings"] = pageSettings
		}

		c.JSON(http.StatusOK, gin.H{"data": data})
	}
}

// GetArcherRegistrationProfile returns a simplified profile for event registration
// It is a public endpoint that requires a valid UUID
// GetArcherRegistrationProfile returns a simplified profile for event registration
// It is a public endpoint that requires a valid UUID
func GetArcherRegistrationProfile(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid := c.Param("uuid")

		if uuid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "UUID is required"})
			return
		}

		var archer struct {
			UUID        string  `json:"uuid" db:"uuid"`
			ID          string  `json:"id" db:"id"`
			FullName    string  `json:"full_name" db:"full_name"`
			Email       *string `json:"email" db:"email"`
			AvatarURL   *string `json:"avatar_url" db:"avatar_url"`
			Gender      *string `json:"gender" db:"gender"`
			DateOfBirth *string `json:"date_of_birth" db:"date_of_birth"`
			Phone       *string `json:"phone" db:"phone"`
			City        *string `json:"city" db:"city"`
			Province    *string `json:"province" db:"province"`
			BowType     *string `json:"bow_type" db:"bow_type"`
			ClubID      *string `json:"club_id" db:"club_id"`
			ClubName    *string `json:"club_name" db:"club_name"`
		}

		query := `
			SELECT 
				a.uuid, a.id, a.full_name, a.email, a.avatar_url,
				a.gender, a.date_of_birth, a.phone,
				a.city, a.province, a.bow_type,
				a.club_id, c.name as club_name
			FROM archers a
			LEFT JOIN clubs c ON a.club_id = c.uuid
			WHERE a.uuid = ?
		`

		err := db.Get(&archer, query, uuid)
		if err != nil {
			logrus.WithError(err).Warnf("Archer registration profile not found: %s", uuid)
			c.JSON(http.StatusNotFound, gin.H{"error": "Archer not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": archer})
	}
}
