package handler

import (
	"net/http"

	"archeryhub-api/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// === OFFICIALS/STAFF MANAGEMENT ===

type Official struct {
	ID           string  `db:"id" json:"id"`
	TournamentID string  `db:"tournament_id" json:"tournament_id"`
	Name         string  `db:"name" json:"name"`
	GivenName    *string `db:"given_name" json:"given_name"`
	Code         *string `db:"code" json:"code"`
	Country      *string `db:"country" json:"country"`
	Role         string  `db:"role" json:"role"`
	CreatedAt    string  `db:"created_at" json:"created_at"`
}

// CreateOfficial adds an official/staff member to a tournament
func CreateOfficial(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("id")

		var req Official
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		officialID := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO tournament_officials 
			(id, tournament_id, name, given_name, code, country, role)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, officialID, tournamentID, req.Name, req.GivenName, req.Code, req.Country, req.Role)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add official"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), tournamentID, "official_added", "official", officialID, "Added official: "+req.Name, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{"id": officialID, "message": "Official added successfully"})
	}
}

// GetOfficials returns all officials for a tournament
func GetOfficials(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("id")

		var officials []Official
		err := db.Select(&officials, `
			SELECT * FROM tournament_officials 
			WHERE tournament_id = ? 
			ORDER BY role ASC, name ASC
		`, tournamentID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch officials"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"officials": officials,
			"total":     len(officials),
		})
	}
}

// UpdateOfficial updates an official's information
func UpdateOfficial(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		officialID := c.Param("officialId")

		var req Official
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := db.Exec(`
			UPDATE tournament_officials SET 
				name = ?, given_name = ?, code = ?, country = ?, role = ?
			WHERE id = ?
		`, req.Name, req.GivenName, req.Code, req.Country, req.Role, officialID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update official"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), "", "official_updated", "official", officialID, "Updated official: "+req.Name, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Official updated successfully"})
	}
}

// DeleteOfficial removes an official from a tournament
func DeleteOfficial(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		officialID := c.Param("officialId")

		_, err := db.Exec("DELETE FROM tournament_officials WHERE id = ?", officialID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete official"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), "", "official_deleted", "official", officialID, "Deleted official", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Official deleted successfully"})
	}
}


// UpdateBackNumber updates the back number and target for a participant
func UpdateBackNumber(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		participantID := c.Param("participantId")

		var req struct {
			BackNumber   string `json:"back_number"`
			TargetNumber string `json:"target_number"`
			Session      int    `json:"session"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := db.Exec(`
			UPDATE tournament_participants 
			SET back_number = ?, target_number = ?, session = ?
			WHERE id = ?
		`, req.BackNumber, req.TargetNumber, req.Session, participantID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update back number"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), "", "back_number_updated", "participant", participantID, "Updated back number/target assignment", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Back number updated successfully"})
	}
}

// GetBackNumbers returns all participants with their assignments
func GetBackNumbers(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("id")

		type ParticipantAssignment struct {
			ID           string  `db:"id" json:"id"`
			AthleteID    string  `db:"athlete_id" json:"athlete_id"`
			AthleteName  string  `db:"athlete_name" json:"athlete_name"`
			EventName    string  `db:"event_name" json:"event_name"`
			BackNumber   *string `db:"back_number" json:"back_number"`
			TargetNumber *string `db:"target_number" json:"target_number"`
			Session      *int    `db:"session" json:"session"`
		}

		var assignments []ParticipantAssignment
		err := db.Select(&assignments, `
			SELECT 
				tp.id, tp.athlete_id, tp.back_number, tp.target_number, tp.session,
				a.full_name as athlete_name,
				CONCAT(d.name, ' - ', c.name) as event_name
			FROM tournament_participants tp
			JOIN archers a ON tp.athlete_id = a.id
			JOIN tournament_events te ON tp.event_id = te.id
			JOIN divisions d ON te.division_id = d.id
			JOIN categories c ON te.category_id = c.id
			WHERE tp.tournament_id = ?
			ORDER BY tp.back_number ASC
		`, tournamentID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch assignments"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"assignments": assignments,
			"total":       len(assignments),
		})
	}
}
