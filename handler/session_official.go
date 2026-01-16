package handler

import (
	"net/http"

	"archeryhub/utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// === SESSION MANAGEMENT ===

type Session struct {
	ID                string  `db:"id" json:"id"`
	TournamentID      string  `db:"tournament_id" json:"tournament_id"`
	SessionOrder      int     `db:"session_order" json:"session_order"`
	Name              string  `db:"name" json:"name"`
	SessionDate       *string `db:"session_date" json:"session_date"`
	StartTime         *string `db:"start_time" json:"start_time"`
	EndTime           *string `db:"end_time" json:"end_time"`
	NumTargets        int     `db:"num_targets" json:"num_targets"`
	AthletesPerTarget int     `db:"athletes_per_target" json:"athletes_per_target"`
	Locked            bool    `db:"locked" json:"locked"`
	Notes             *string `db:"notes" json:"notes"`
}

// CreateSession creates a new session for a tournament
func CreateSession(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("id")

		var req Session
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		sessionID := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO sessions 
			(id, tournament_id, session_order, name, session_date, start_time, end_time, 
			 num_targets, athletes_per_target, locked, notes)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, sessionID, tournamentID, req.SessionOrder, req.Name, req.SessionDate,
			req.StartTime, req.EndTime, req.NumTargets, req.AthletesPerTarget,
			req.Locked, req.Notes)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), tournamentID, "session_created", "session", sessionID, "Created session: "+req.Name, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{"id": sessionID, "message": "Session created successfully"})
	}
}

// GetSessions returns all sessions for a tournament
func GetSessions(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("id")

		var sessions []Session
		err := db.Select(&sessions, `
			SELECT * FROM sessions 
			WHERE tournament_id = ? 
			ORDER BY session_order ASC
		`, tournamentID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sessions"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"sessions": sessions,
			"total":    len(sessions),
		})
	}
}

// UpdateSession updates a session
func UpdateSession(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("sessionId")

		var req Session
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := db.Exec(`
			UPDATE sessions SET 
				session_order = ?, name = ?, session_date = ?, start_time = ?, 
				end_time = ?, num_targets = ?, athletes_per_target = ?, locked = ?, notes = ?
			WHERE id = ?
		`, req.SessionOrder, req.Name, req.SessionDate, req.StartTime, req.EndTime,
			req.NumTargets, req.AthletesPerTarget, req.Locked, req.Notes, sessionID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), "", "session_updated", "session", sessionID, "Updated session", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Session updated successfully"})
	}
}

// DeleteSession deletes a session
func DeleteSession(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("sessionId")

		_, err := db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete session"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), "", "session_deleted", "session", sessionID, "Deleted session", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Session deleted successfully"})
	}
}

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

// === DISTANCE MANAGEMENT ===

type Distance struct {
	ID            string  `db:"id" json:"id"`
	TournamentID  string  `db:"tournament_id" json:"tournament_id"`
	EventID       string  `db:"event_id" json:"event_id"`
	DistanceOrder int     `db:"distance_order" json:"distance_order"`
	DistanceValue int     `db:"distance_value" json:"distance_value"`
	ArrowsPerEnd  int     `db:"arrows_per_end" json:"arrows_per_end"`
	NumEnds       int     `db:"num_ends" json:"num_ends"`
	TargetFace    *string `db:"target_face" json:"target_face"`
}

// CreateDistance creates a distance configuration for an event
func CreateDistance(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req Distance
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		distanceID := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO distances 
			(id, tournament_id, event_id, distance_order, distance_value, arrows_per_end, num_ends, target_face)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, distanceID, req.TournamentID, req.EventID, req.DistanceOrder, req.DistanceValue,
			req.ArrowsPerEnd, req.NumEnds, req.TargetFace)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create distance"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), req.TournamentID, "distance_created", "distance", distanceID, "Created distance configuration", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{"id": distanceID, "message": "Distance created successfully"})
	}
}

// GetDistances returns all distances for an event or tournament
func GetDistances(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("id")
		eventID := c.Query("event_id")

		query := "SELECT * FROM distances WHERE tournament_id = ?"
		args := []interface{}{tournamentID}

		if eventID != "" {
			query += " AND event_id = ?"
			args = append(args, eventID)
		}

		query += " ORDER BY event_id, distance_order ASC"

		var distances []Distance
		err := db.Select(&distances, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch distances"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"distances": distances,
			"total":     len(distances),
		})
	}
}

// UpdateDistance updates a distance configuration
func UpdateDistance(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		distanceID := c.Param("distanceId")

		var req Distance
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := db.Exec(`
			UPDATE distances SET 
				distance_order = ?, distance_value = ?, arrows_per_end = ?, num_ends = ?, target_face = ?
			WHERE id = ?
		`, req.DistanceOrder, req.DistanceValue, req.ArrowsPerEnd, req.NumEnds, req.TargetFace, distanceID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update distance"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), "", "distance_updated", "distance", distanceID, "Updated distance configuration", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Distance updated successfully"})
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
				CONCAT(a.first_name, ' ', a.last_name) as athlete_name,
				CONCAT(d.name, ' - ', c.name) as event_name
			FROM tournament_participants tp
			JOIN athletes a ON tp.athlete_id = a.id
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
