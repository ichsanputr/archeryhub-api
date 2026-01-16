package handler

import (
	"fmt"
	"net/http"
	"time"

	"archeryhub-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Accreditation represents an athlete's accreditation status
type Accreditation struct {
	ID            string  `json:"id" db:"id"`
	TournamentID  string  `json:"tournament_id" db:"tournament_id"`
	ParticipantID string  `json:"participant_id" db:"participant_id"`
	CardNumber    string  `json:"card_number" db:"card_number"`
	CardType      string  `json:"card_type" db:"card_type"` // athlete, coach, official, media, vip
	Status        string  `json:"status" db:"status"` // pending, printed, issued, revoked
	PrintedAt     *string `json:"printed_at" db:"printed_at"`
	IssuedAt      *string `json:"issued_at" db:"issued_at"`
	AccessAreas   string  `json:"access_areas" db:"access_areas"` // CSV of allowed areas
	CreatedAt     string  `json:"created_at" db:"created_at"`
}

// AccreditationWithDetails includes participant info
type AccreditationWithDetails struct {
	Accreditation
	FirstName  string  `json:"first_name" db:"first_name"`
	LastName   string  `json:"last_name" db:"last_name"`
	Country    *string `json:"country" db:"country"`
	BackNumber *string `json:"back_number" db:"back_number"`
	PhotoURL   *string `json:"photo_url" db:"photo_url"`
}

// CreateAccreditation creates a new accreditation record
func CreateAccreditation(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("tournamentId")
		userID, _ := c.Get("user_id")

		var req struct {
			ParticipantID string `json:"participant_id" binding:"required"`
			CardType      string `json:"card_type" binding:"required,oneof=athlete coach official media vip"`
			AccessAreas   string `json:"access_areas"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		accredID := uuid.New().String()
		cardNumber := generateCardNumber(tournamentID, req.CardType)

		_, err := db.Exec(`
			INSERT INTO accreditations 
			(id, tournament_id, participant_id, card_number, card_type, status, access_areas)
			VALUES (?, ?, ?, ?, ?, 'pending', ?)
		`, accredID, tournamentID, req.ParticipantID, cardNumber, req.CardType, req.AccessAreas)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create accreditation"})
			return
		}

		utils.LogActivity(db, userID.(string), tournamentID, "accreditation_created", "accreditation", accredID,
			fmt.Sprintf("Created accreditation for participant"), c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"id":          accredID,
			"card_number": cardNumber,
			"message":     "Accreditation created successfully",
		})
	}
}

func generateCardNumber(tournamentID, cardType string) string {
	prefix := "A" // Athlete
	switch cardType {
	case "coach":
		prefix = "C"
	case "official":
		prefix = "O"
	case "media":
		prefix = "M"
	case "vip":
		prefix = "V"
	}
	return fmt.Sprintf("%s-%s-%d", prefix, tournamentID[:8], time.Now().UnixNano()%10000)
}

// GetAccreditations returns all accreditations for a tournament
func GetAccreditations(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("tournamentId")
		status := c.Query("status")
		cardType := c.Query("card_type")

		query := `
			SELECT acc.*, a.first_name, a.last_name, a.country, tp.back_number, a.photo_url
			FROM accreditations acc
			JOIN tournament_participants tp ON acc.participant_id = tp.id
			JOIN athletes a ON tp.athlete_id = a.id
			WHERE acc.tournament_id = ?
		`
		args := []interface{}{tournamentID}

		if status != "" {
			query += " AND acc.status = ?"
			args = append(args, status)
		}

		if cardType != "" {
			query += " AND acc.card_type = ?"
			args = append(args, cardType)
		}

		query += " ORDER BY a.last_name, a.first_name"

		var accreditations []AccreditationWithDetails
		err := db.Select(&accreditations, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch accreditations"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"accreditations": accreditations,
			"total":          len(accreditations),
		})
	}
}

// UpdateAccreditationStatus updates the status of an accreditation
func UpdateAccreditationStatus(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		accredID := c.Param("accredId")
		userID, _ := c.Get("user_id")

		var req struct {
			Status string `json:"status" binding:"required,oneof=pending printed issued revoked"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updateQuery := "UPDATE accreditations SET status = ?"
		args := []interface{}{req.Status}

		if req.Status == "printed" {
			updateQuery += ", printed_at = NOW()"
		} else if req.Status == "issued" {
			updateQuery += ", issued_at = NOW()"
		}

		updateQuery += " WHERE id = ?"
		args = append(args, accredID)

		result, err := db.Exec(updateQuery, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Accreditation not found"})
			return
		}

		var tournamentID string
		db.Get(&tournamentID, "SELECT tournament_id FROM accreditations WHERE id = ?", accredID)

		utils.LogActivity(db, userID.(string), tournamentID, "accreditation_status_updated", "accreditation", accredID,
			fmt.Sprintf("Status updated to %s", req.Status), c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
	}
}

// GateCheck validates an accreditation at a gate
type GateCheckLog struct {
	ID            string `json:"id" db:"id"`
	TournamentID  string `json:"tournament_id" db:"tournament_id"`
	AccredID      string `json:"accreditation_id" db:"accreditation_id"`
	GateName      string `json:"gate_name" db:"gate_name"`
	Direction     string `json:"direction" db:"direction"` // in, out
	CheckedAt     string `json:"checked_at" db:"checked_at"`
	CheckedBy     string `json:"checked_by" db:"checked_by"`
	AccessGranted bool   `json:"access_granted" db:"access_granted"`
	Reason        string `json:"reason" db:"reason"`
}

// GateCheck processes a gate access attempt
func GateCheck(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var req struct {
			CardNumber string `json:"card_number" binding:"required"`
			GateName   string `json:"gate_name" binding:"required"`
			Direction  string `json:"direction" binding:"required,oneof=in out"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Find accreditation
		var accred Accreditation
		err := db.Get(&accred, "SELECT * FROM accreditations WHERE card_number = ?", req.CardNumber)
		if err != nil {
			logID := uuid.New().String()
			db.Exec(`INSERT INTO gate_check_logs (id, gate_name, direction, checked_by, access_granted, reason) 
				VALUES (?, ?, ?, ?, false, 'Card not found')`, logID, req.GateName, req.Direction, userID.(string))
			c.JSON(http.StatusNotFound, gin.H{"access": false, "reason": "Card not found"})
			return
		}

		// Check if card is valid
		accessGranted := accred.Status == "issued"
		reason := ""
		if !accessGranted {
			reason = "Card not issued"
		}

		// Log the gate check
		logID := uuid.New().String()
		db.Exec(`INSERT INTO gate_check_logs 
			(id, tournament_id, accreditation_id, gate_name, direction, checked_by, access_granted, reason) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			logID, accred.TournamentID, accred.ID, req.GateName, req.Direction, userID.(string), accessGranted, reason)

		c.JSON(http.StatusOK, gin.H{
			"access":      accessGranted,
			"reason":      reason,
			"card_type":   accred.CardType,
			"card_number": accred.CardNumber,
		})
	}
}

// GetGateSituation returns current gate access statistics
func GetGateSituation(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("tournamentId")

		type GateStat struct {
			GateName   string `json:"gate_name" db:"gate_name"`
			InCount    int    `json:"in_count" db:"in_count"`
			OutCount   int    `json:"out_count" db:"out_count"`
			CurrentIn  int    `json:"current_in" db:"current_in"`
		}

		var stats []GateStat
		db.Select(&stats, `
			SELECT 
				gate_name,
				SUM(CASE WHEN direction = 'in' THEN 1 ELSE 0 END) as in_count,
				SUM(CASE WHEN direction = 'out' THEN 1 ELSE 0 END) as out_count,
				SUM(CASE WHEN direction = 'in' THEN 1 ELSE -1 END) as current_in
			FROM gate_check_logs
			WHERE tournament_id = ? AND access_granted = true
			GROUP BY gate_name
		`, tournamentID)

		c.JSON(http.StatusOK, gin.H{
			"gates": stats,
			"total": len(stats),
		})
	}
}

// BulkCreateAccreditations creates accreditations for all participants in a tournament
func BulkCreateAccreditations(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("tournamentId")
		userID, _ := c.Get("user_id")

		var req struct {
			CardType    string `json:"card_type" binding:"required"`
			AccessAreas string `json:"access_areas"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get all participants without accreditation
		var participants []string
		db.Select(&participants, `
			SELECT tp.id FROM tournament_participants tp
			LEFT JOIN accreditations acc ON tp.id = acc.participant_id
			WHERE tp.tournament_id = ? AND acc.id IS NULL
		`, tournamentID)

		created := 0
		for _, pid := range participants {
			accredID := uuid.New().String()
			cardNumber := generateCardNumber(tournamentID, req.CardType)

			_, err := db.Exec(`
				INSERT INTO accreditations 
				(id, tournament_id, participant_id, card_number, card_type, status, access_areas)
				VALUES (?, ?, ?, ?, ?, 'pending', ?)
			`, accredID, tournamentID, pid, cardNumber, req.CardType, req.AccessAreas)

			if err == nil {
				created++
			}
		}

		utils.LogActivity(db, userID.(string), tournamentID, "bulk_accreditation", "accreditation", "",
			fmt.Sprintf("Created %d accreditations", created), c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"message": "Bulk accreditations created",
			"created": created,
		})
	}
}
