package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ClubMember represents a club membership
type ClubMember struct {
	UUID      string     `json:"uuid" db:"uuid"`
	ClubID    string     `json:"club_id" db:"club_id"`
	ArcherID  string     `json:"archer_id" db:"archer_id"`
	Status    string     `json:"status" db:"status"`
	Role      string     `json:"role" db:"role"`
	JoinedAt  *time.Time `json:"joined_at" db:"joined_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// GetClubs returns all clubs (public)
func GetClubs(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := `
			SELECT c.*, 
				(SELECT COUNT(*) FROM club_members WHERE club_id = c.uuid AND status = 'active') as member_count
			FROM clubs c 
			WHERE c.is_verified = true OR c.status = 'active'
			ORDER BY c.name ASC
		`
		
		var clubs []struct {
			UUID        string  `json:"uuid" db:"uuid"`
			Name        string  `json:"name" db:"name"`
			Slug        string  `json:"slug" db:"slug"`
			LogoURL     *string `json:"logo_url" db:"logo_url"`
			BannerURL   *string `json:"banner_url" db:"banner_url"`
			City        *string `json:"city" db:"city"`
			Province    *string `json:"province" db:"province"`
			IsVerified  bool    `json:"verified" db:"is_verified"`
			MemberCount int     `json:"member_count" db:"member_count"`
		}
		
		err := db.Select(&clubs, query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch clubs"})
			return
		}
		
		if clubs == nil {
			clubs = make([]struct {
				UUID        string  `json:"uuid" db:"uuid"`
				Name        string  `json:"name" db:"name"`
				Slug        string  `json:"slug" db:"slug"`
				LogoURL     *string `json:"logo_url" db:"logo_url"`
				BannerURL   *string `json:"banner_url" db:"banner_url"`
				City        *string `json:"city" db:"city"`
				Province    *string `json:"province" db:"province"`
				IsVerified  bool    `json:"verified" db:"is_verified"`
				MemberCount int     `json:"member_count" db:"member_count"`
			}, 0)
		}
		
		c.JSON(http.StatusOK, gin.H{"data": clubs})
	}
}

// GetClubBySlug returns a single club by slug
func GetClubBySlug(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")
		
		var club struct {
			UUID        string  `json:"uuid" db:"uuid"`
			Name        string  `json:"name" db:"name"`
			Slug        string  `json:"slug" db:"slug"`
			Description *string `json:"description" db:"description"`
			LogoURL     *string `json:"logo_url" db:"logo_url"`
			BannerURL   *string `json:"banner_url" db:"banner_url"`
			Address     *string `json:"address" db:"address"`
			City        *string `json:"city" db:"city"`
			Province    *string `json:"province" db:"province"`
			Phone       *string `json:"phone" db:"phone"`
			Email       *string `json:"email" db:"email"`
			Website     *string `json:"website" db:"website"`
			IsVerified  bool    `json:"verified" db:"is_verified"`
			CreatedAt   string  `json:"created_at" db:"created_at"`
		}
		
		err := db.Get(&club, "SELECT * FROM clubs WHERE slug = ? OR uuid = ?", slug, slug)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Club not found"})
			return
		}
		
		// Get member count
		var memberCount int
		db.Get(&memberCount, "SELECT COUNT(*) FROM club_members WHERE club_id = ? AND status = 'active'", club.UUID)
		
		c.JSON(http.StatusOK, gin.H{
			"data": club,
			"member_count": memberCount,
		})
	}
}

// JoinClub allows an archer to request membership
func JoinClub(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := c.Param("id")
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")
		
		// Only archers can join clubs
		if userType != "archer" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only archers can join clubs"})
			return
		}
		
		// Check if club exists
		var clubExists bool
		err := db.Get(&clubExists, "SELECT EXISTS(SELECT 1 FROM clubs WHERE uuid = ?)", clubID)
		if err != nil || !clubExists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Club not found"})
			return
		}
		
		// Check if already a member of any club
		var existingMembership string
		err = db.Get(&existingMembership, "SELECT club_id FROM club_members WHERE archer_id = ? AND status IN ('pending', 'active')", userID)
		if err == nil && existingMembership != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "You are already a member or have a pending request to another club"})
			return
		}
		
		// Create membership request
		memberID := uuid.New().String()
		_, err = db.Exec(`
			INSERT INTO club_members (uuid, club_id, archer_id, status, role)
			VALUES (?, ?, ?, 'pending', 'member')
		`, memberID, clubID, userID)
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit membership request"})
			return
		}
		
		c.JSON(http.StatusCreated, gin.H{
			"message": "Membership request submitted successfully",
			"id": memberID,
		})
	}
}

// GetMyClubMembership returns the current user's club membership status
func GetMyClubMembership(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		
		var membership struct {
			ClubMember
			ClubName string `json:"club_name" db:"club_name"`
		}
		
		err := db.Get(&membership, `
			SELECT cm.*, c.name as club_name 
			FROM club_members cm 
			JOIN clubs c ON cm.club_id = c.uuid 
			WHERE cm.archer_id = ? AND cm.status IN ('pending', 'active')
		`, userID)
		
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"data": nil, "message": "No active membership"})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{"data": membership})
	}
}

// LeaveClub allows an archer to leave their club
func LeaveClub(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		
		result, err := db.Exec(`
			UPDATE club_members SET status = 'left', updated_at = NOW() 
			WHERE archer_id = ? AND status = 'active'
		`, userID)
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to leave club"})
			return
		}
		
		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No active membership found"})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{"message": "Successfully left the club"})
	}
}

// ApproveClubMember allows club admin to approve a membership request
func ApproveClubMember(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		memberID := c.Param("memberId")
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")
		
		// Only club admins can approve
		if userType != "club" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only club owners can approve members"})
			return
		}
		
		// Verify the member belongs to the user's club
		var clubID string
		err := db.Get(&clubID, "SELECT uuid FROM clubs WHERE owner_id = ?", userID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't own any club"})
			return
		}
		
		now := time.Now()
		result, err := db.Exec(`
			UPDATE club_members SET status = 'active', joined_at = ?, updated_at = NOW() 
			WHERE uuid = ? AND club_id = ? AND status = 'pending'
		`, now, memberID, clubID)
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve member"})
			return
		}
		
		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Membership request not found or already processed"})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{"message": "Member approved successfully"})
	}
}

// GetClubMembers returns all members of a club
func GetClubMembers(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := c.Param("id")
		
		var members []struct {
			ClubMember
			ArcherName string `json:"archer_name" db:"archer_name"`
		}
		
		err := db.Select(&members, `
			SELECT cm.*, u.full_name as archer_name
			FROM club_members cm
			JOIN users u ON cm.archer_id = u.uuid
			WHERE cm.club_id = ?
			ORDER BY cm.status ASC, cm.created_at DESC
		`, clubID)
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch members"})
			return
		}
		
		if members == nil {
			members = make([]struct {
				ClubMember
				ArcherName string `json:"archer_name" db:"archer_name"`
			}, 0)
		}
		
		c.JSON(http.StatusOK, gin.H{"data": members})
	}
}
