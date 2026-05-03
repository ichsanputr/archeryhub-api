package mobile

import (
	"archeryhub-api/utils"
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type BoardInfo struct {
	UUID         string `db:"uuid"`
	SessionUUID  string `db:"session_uuid"`
	CategoryUUID string `db:"category_uuid"`
	BoardNumber  int    `db:"board_number"`
	Code         string `db:"code"`
	SessionName  string `db:"session_name"`
	EventUUID    string `db:"event_uuid"`
	EventName    string `db:"event_name"`
}

type ArcherInfo struct {
	AssignmentUUID  string  `db:"assignment_uuid" json:"assignment_uuid"`
	ParticipantUUID string  `db:"participant_uuid" json:"participant_uuid"`
	Position        string  `db:"position" json:"position"`       // e.g. "A"
	TargetName      string  `db:"target_name" json:"target_name"` // e.g. "003A"
	Name            string  `db:"name" json:"name"`
	Division        string  `db:"division" json:"division"`
	AvatarURL       *string `db:"avatar_url" json:"avatar_url"`
	CurrentScore    int     `db:"current_score" json:"current_score"`
	EndsCompleted   int     `db:"ends_completed" json:"ends_completed"`
	TotalEnds       int     `db:"total_ends" json:"total_ends"`
}

type ElimBoardInfo struct {
	UUID         string `db:"uuid"`
	BracketUUID  string `db:"bracket_uuid"`
	CategoryUUID string `db:"category_uuid"`
	BoardNumber  int    `db:"board_number"`
	Code         string `db:"code"`
	CategoryName string `db:"category_name"`
	EventUUID    string `db:"event_uuid"`
	EventName    string `db:"event_name"`
}

type ElimArcherInfo struct {
	MatchUUID  string  `json:"match_uuid"`
	MatchID    string  `json:"match_id"`
	Side       string  `json:"side"` // A or B
	TargetName string  `json:"target_name"`
	Name       string  `json:"name"`
	Club       string  `json:"club"`
	AvatarURL  *string `json:"avatar_url"`
	Score      int     `json:"score"`
	Status     string  `json:"status"`
}

type MatchRow struct {
	MatchUUID  string         `db:"match_uuid"`
	MatchID    sql.NullString `db:"match_id"`
	TargetName string         `db:"target_name"`
	RoundNo    int            `db:"round_no"`
	MatchNo    int            `db:"match_no"`
	Status     string         `db:"status"`
	NameA      sql.NullString `db:"name_a"`
	AvatarA    sql.NullString `db:"avatar_a"`
	ClubA      sql.NullString `db:"club_a"`
	ScoreA     int            `db:"score_a"`
	NameB      sql.NullString `db:"name_b"`
	AvatarB    sql.NullString `db:"avatar_b"`
	ClubB      sql.NullString `db:"club_b"`
	ScoreB     int            `db:"score_b"`
}

// MobileScanTarget handles scanning a target board QR code
// @Summary Scan Target QR
// @Description Scan a target board QR code to get participant info for scoring. Returns qualification or elimination data based on the QR.
// @Tags Mobile - Scorekeeper
// @Produce json
// @Security ApiKeyAuth
// @Param code query string true "Target Board QR Code"
// @Success 200 {object} MobileScanTargetQualificationResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /mobile/scan [get]
func MobileScanTarget(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Query("code")
		if code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "kode wajib diisi"})
			return
		}


		var board BoardInfo
		err := db.Get(&board, `
			SELECT 
				tbq.uuid, tbq.session_uuid, tbq.category_uuid, tbq.board_number, tbq.code,
				qs.name as session_name, qs.event_uuid,
				e.name as event_name
			FROM target_board_qualification tbq
			JOIN qualification_sessions qs ON tbq.session_uuid = qs.uuid
			JOIN events e ON qs.event_uuid = e.uuid
			WHERE tbq.code = ?
			LIMIT 1
		`, code)

		if err == nil {
			// --- Handle Qualification Board ---

			var archers []ArcherInfo
			err = db.Select(&archers, `
				SELECT
					qta.uuid as assignment_uuid,
				qta.participant_uuid,
				RIGHT(et.target_name, 1) as position,
				et.target_name as target_name,
				COALESCE(a.full_name, '') as name,
				COALESCE(CONCAT(bt.name, ' ', ag.name), '') as division,
					a.avatar_url,
					COALESCE(SUM(qes.total_score_end), 0) as current_score,
					COUNT(qes.uuid) as ends_completed,
					qs.total_ends
				FROM qualification_target_assignments qta
				JOIN event_targets et ON qta.target_uuid = et.uuid
				JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
				JOIN event_participants ep ON qta.participant_uuid = ep.uuid
				LEFT JOIN archers a ON ep.archer_id = a.uuid
				LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
				LEFT JOIN ref_bow_types bt ON ec.division_uuid = bt.uuid
				LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
				LEFT JOIN qualification_end_scores qes ON qes.participant_uuid = ep.uuid AND qes.session_uuid = qta.session_uuid
				WHERE qta.target_board_id = ? AND qta.session_uuid = ?
				GROUP BY qta.uuid, qta.participant_uuid, et.target_name, a.full_name, bt.name, ag.name, qs.total_ends, a.avatar_url
				ORDER BY et.target_name ASC
			`, board.UUID, board.SessionUUID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data atlet", "details": err.Error()})
				return
			}

			for i := range archers {
				if archers[i].AvatarURL != nil {
					masked := utils.MaskMediaURL(*archers[i].AvatarURL)
					archers[i].AvatarURL = &masked
				}
			}

			c.JSON(http.StatusOK, MobileScanTargetQualificationResponse{
				Type: "qualification",
				Board: MobileScannedBoard{
					UUID:         board.UUID,
					BoardNumber:  board.BoardNumber,
					Code:         board.Code,
					SessionUUID:  board.SessionUUID,
					SessionName:  board.SessionName,
					EventUUID:    board.EventUUID,
					EventName:    board.EventName,
					CategoryUUID: board.CategoryUUID,
				},
				Archers: castToScannedArchersQual(archers),
			})
			return
		}

		// --- If not found in Qual, try Elimination ---
		var eb ElimBoardInfo
		err = db.Get(&eb, `
			SELECT 
				tbe.uuid, tbe.bracket_uuid, tbe.category_uuid, tbe.board_number, tbe.code,
				COALESCE(ec.category_name_custom, '') as category_name,
				e.uuid as event_uuid, e.name as event_name
			FROM target_board_elimination tbe
			JOIN elimination_brackets eb ON tbe.bracket_uuid = eb.uuid
			JOIN events e ON eb.event_uuid = e.uuid
			LEFT JOIN event_categories ec ON tbe.category_uuid = ec.uuid
			WHERE tbe.code = ?
			LIMIT 1
		`, code)

		if err == nil {
			// Fetch matches on this elimination board
			var rows []MatchRow
			err = db.Select(&rows, `
				SELECT 
					em.uuid AS match_uuid, em.match_id, et.target_name, em.round_no, em.match_no, em.status,
					COALESCE(aA.full_name, tA.team_name, '') as name_a, aA.avatar_url as avatar_a, COALESCE(cA.name, '') as club_a,
					COALESCE(em.total_score_a, em.total_points_a, 0) as score_a,
					COALESCE(aB.full_name, tB.team_name, '') as name_b, aB.avatar_url as avatar_b, COALESCE(cB.name, '') as club_b,
					COALESCE(em.total_score_b, em.total_points_b, 0) as score_b
				FROM elimination_matches em
				JOIN event_targets et ON em.target_uuid = et.uuid
				LEFT JOIN elimination_entries eeA ON em.entry_a_uuid = eeA.uuid
				LEFT JOIN archers aA ON eeA.participant_type = 'archer' AND eeA.participant_uuid = aA.uuid
				LEFT JOIN clubs cA ON aA.club_id = cA.uuid
				LEFT JOIN teams tA ON eeA.participant_type = 'team' AND eeA.participant_uuid = tA.uuid
				LEFT JOIN elimination_entries eeB ON em.entry_b_uuid = eeB.uuid
				LEFT JOIN archers aB ON eeB.participant_type = 'archer' AND eeB.participant_uuid = aB.uuid
				LEFT JOIN clubs cB ON aB.club_id = cB.uuid
				LEFT JOIN teams tB ON eeB.participant_type = 'team' AND eeB.participant_uuid = tB.uuid
				WHERE em.bracket_uuid = ? AND et.board_number = ?
				ORDER BY em.round_no DESC, em.match_no ASC
			`, eb.BracketUUID, eb.BoardNumber)

			archers := []ElimArcherInfo{}
			for _, r := range rows {
				// Side A
				a := ElimArcherInfo{
					MatchUUID:  r.MatchUUID,
					MatchID:    r.MatchID.String,
					Side:       "A",
					TargetName: r.TargetName,
					Name:       r.NameA.String,
					Club:       r.ClubA.String,
					Score:      r.ScoreA,
					Status:     r.Status,
				}
				if r.AvatarA.Valid {
					masked := utils.MaskMediaURL(r.AvatarA.String)
					a.AvatarURL = &masked
				}
				archers = append(archers, a)

				// Side B
				b := ElimArcherInfo{
					MatchUUID:  r.MatchUUID,
					MatchID:    r.MatchID.String,
					Side:       "B",
					TargetName: r.TargetName,
					Name:       r.NameB.String,
					Club:       r.ClubB.String,
					Score:      r.ScoreB,
					Status:     r.Status,
				}
				if r.AvatarB.Valid {
					masked := utils.MaskMediaURL(r.AvatarB.String)
					b.AvatarURL = &masked
				}
				archers = append(archers, b)
			}

			c.JSON(http.StatusOK, MobileScanTargetEliminationResponse{
				Type: "elimination",
				Board: MobileScannedBoard{
					UUID:          eb.UUID,
					BoardNumber:   eb.BoardNumber,
					Code:          eb.Code,
					BracketUUID:   eb.BracketUUID,
					CategoryName:  eb.CategoryName,
					EventUUID:     eb.EventUUID,
					EventName:     eb.EventName,
				},
				Archers: castToScannedArchersElim(archers),
			})
			return
		}

		c.JSON(http.StatusNotFound, gin.H{"error": "Kode tidak ditemukan", "code": "not_found"})
	}
}

// MobileGetSessionBoards returns all target boards in a qualification session
// with each archer's current score summary Î“Ã‡Ã¶ powers the "List Targets" leaderboard screen.
//
// MobileGetSessionBoards returns leaderboard for a session
// @Summary List Session Targets
// @Description Get all target boards and current scores for a qualification session
// @Tags Mobile - Scorekeeper
// @Produce json
// @Security ApiKeyAuth
// @Param session_id query string true "Session UUID"
// @Success 200 {object} MobileSessionBoardsResponse
// @Router /mobile/sessions/boards [get]
func MobileGetSessionBoards(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Query("session_id")
		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id wajib diisi"})
			return
		}

		type SessionInfo struct {
			UUID         string `db:"uuid"`
			Name         string `db:"name"`
			EventUUID    string `db:"event_uuid"`
			EventName    string `db:"event_name"`
			TotalEnds    int    `db:"total_ends"`
			ArrowsPerEnd int    `db:"arrows_per_end"`
		}

		var session SessionInfo
		err := db.Get(&session, `
			SELECT qs.uuid, qs.name, qs.event_uuid, e.name as event_name, qs.total_ends, qs.arrows_per_end
			FROM qualification_sessions qs
			JOIN events e ON qs.event_uuid = e.uuid
			WHERE qs.uuid = ?
		`, sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sesi tidak ditemukan"})
			return
		}

		type ArcherAtBoard struct {
			AssignmentUUID  string  `db:"assignment_uuid" json:"assignment_uuid"`
			ParticipantUUID string  `db:"participant_uuid" json:"participant_uuid"`
			TargetName      string  `db:"target_name" json:"target_name"`
			BoardNumber     int     `db:"board_number" json:"board_number"`
			Position        string  `db:"position" json:"position"`
			Name            string  `db:"name" json:"name"`
			Division        string  `db:"division" json:"division"`
			AvatarURL       *string `db:"avatar_url" json:"avatar_url"`
			CurrentScore    int     `db:"current_score" json:"current_score"`
			EndsCompleted   int     `db:"ends_completed" json:"ends_completed"`
			LastEndScore    int     `db:"last_end_score" json:"last_end_score"`
			Rank            int     `json:"rank,omitempty"`
		}

		var archers []ArcherAtBoard
		err = db.Select(&archers, `
			SELECT
				qta.uuid as assignment_uuid,
				qta.participant_uuid,
				et.target_name,
				et.board_number,
				RIGHT(et.target_name, 1) as position,
				COALESCE(a.full_name, '') as name,
				COALESCE(CONCAT(bt.name, ' ', ag.name), '') as division,
				a.avatar_url,
				COALESCE(SUM(qes.total_score_end), 0) as current_score,
				COUNT(qes.uuid) as ends_completed,
				COALESCE((
					SELECT qes2.total_score_end
					FROM qualification_end_scores qes2
					WHERE qes2.participant_uuid = ep.uuid AND qes2.session_uuid = qta.session_uuid
					ORDER BY qes2.end_number DESC
					LIMIT 1
				), 0) as last_end_score
			FROM qualification_target_assignments qta
			JOIN event_targets et ON qta.target_uuid = et.uuid
			JOIN event_participants ep ON qta.participant_uuid = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types bt ON ec.division_uuid = bt.uuid
			LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
			LEFT JOIN qualification_end_scores qes ON qes.participant_uuid = ep.uuid AND qes.session_uuid = qta.session_uuid
			WHERE qta.session_uuid = ?
			GROUP BY qta.uuid, qta.participant_uuid, et.target_name, et.board_number, a.full_name, bt.name, ag.name, a.avatar_url, ep.uuid
			ORDER BY et.board_number ASC, et.target_name ASC
		`, sessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data bantalan", "details": err.Error()})
			return
		}

		// Assign rank by current_score desc
		for i := range archers {
			if archers[i].AvatarURL != nil {
				masked := utils.MaskMediaURL(*archers[i].AvatarURL)
				archers[i].AvatarURL = &masked
			}
			archers[i].Rank = i + 1
		}

		c.JSON(http.StatusOK, gin.H{
			"session": gin.H{
				"uuid":           session.UUID,
				"name":           session.Name,
				"event_uuid":     session.EventUUID,
				"event_name":     session.EventName,
				"total_ends":     session.TotalEnds,
				"arrows_per_end": session.ArrowsPerEnd,
			},
			"archers": archers,
		})
	}
}

// MobileGetAssignmentScoreDetail returns the full arrow-by-arrow score history for a
// participant's target assignment Î“Ã‡Ã¶ powers the "Detail Score Target" screen.
//
// MobileGetAssignmentScoreDetail godoc

type ArrowScore struct {
	ArrowNumber int  `db:"arrow_number" json:"arrow_number"`
	Score       int  `db:"score" json:"score"`
	IsX         bool `db:"is_x" json:"is_x"`
}

type EndScore struct {
	EndNumber    int          `db:"end_number" json:"end_number"`
	EndScoreUUID string       `db:"end_score_uuid" json:"end_score_uuid"`
	EndTotal     int          `db:"end_total" json:"end_total"`
	XCount       int          `db:"x_count" json:"x_count"`
	TenCount     int          `db:"ten_count" json:"ten_count"`
	CumTotal     int          `json:"cumulative_total"`
	Arrows       []ArrowScore `json:"arrows"`
}

// MobileGetAssignmentScoreDetail returns detailed scores for an assignment
// @Summary Get Assignment Score Detail
// @Description Get arrow-by-arrow score history for a specific participant assignment
// @Tags Mobile - Scorekeeper
// @Produce json
// @Security ApiKeyAuth
// @Param assignmentId path string true "Assignment UUID"
// @Success 200 {object} MobileAssignmentScoreDetailResponse
// @Router /mobile/assignments/{assignmentId}/detail [get]
func MobileGetAssignmentScoreDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		assignmentID := c.Param("assignmentId")
		if assignmentID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "assignmentId wajib diisi"})
			return
		}

		type AssignmentMeta struct {
			AssignmentUUID  string  `db:"assignment_uuid"`
			ParticipantUUID string  `db:"participant_uuid"`
			SessionUUID     string  `db:"session_uuid"`
			SessionName     string  `db:"session_name"`
			EventName       string  `db:"event_name"`
			TargetName      string  `db:"target_name"`
			ArcherName      string  `db:"archer_name"`
			Division        string  `db:"division"`
			AvatarURL       *string `db:"avatar_url"`
			TotalEnds       int     `db:"total_ends"`
			ArrowsPerEnd    int     `db:"arrows_per_end"`
		}

		var meta AssignmentMeta
		err := db.Get(&meta, `
			SELECT
				qta.uuid as assignment_uuid,
				qta.participant_uuid,
				qta.session_uuid,
				qs.name as session_name,
				e.name as event_name,
				et.target_name,
				COALESCE(a.full_name, '') as archer_name,
				COALESCE(CONCAT(bt.name, ' ', ag.name), '') as division,
				a.avatar_url,
				qs.total_ends,
				qs.arrows_per_end
			FROM qualification_target_assignments qta
			JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
			JOIN events e ON qs.event_uuid = e.uuid
			JOIN event_targets et ON qta.target_uuid = et.uuid
			JOIN event_participants ep ON qta.participant_uuid = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types bt ON ec.division_uuid = bt.uuid
			LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
			WHERE qta.uuid = ?
			LIMIT 1
		`, assignmentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Data pendaftaran tidak ditemukan"})
			return
		}

		if meta.AvatarURL != nil {
			masked := utils.MaskMediaURL(*meta.AvatarURL)
			meta.AvatarURL = &masked
		}


		var ends []EndScore
		err = db.Select(&ends, `
			SELECT 
				qes.end_number,
				qes.uuid as end_score_uuid,
				qes.total_score_end as end_total,
				qes.x_count_end as x_count,
				qes.ten_count_end as ten_count
			FROM qualification_end_scores qes
			WHERE qes.participant_uuid = ? AND qes.session_uuid = ?
			ORDER BY qes.end_number ASC
		`, meta.ParticipantUUID, meta.SessionUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil skor babak"})
			return
		}

		// For each end, fetch arrow scores and compute cumulative total
		cumTotal := 0
		totalXCount := 0
		totalTenPlusXCount := 0

		for i := range ends {
			cumTotal += ends[i].EndTotal
			ends[i].CumTotal = cumTotal
			totalXCount += ends[i].XCount
			totalTenPlusXCount += ends[i].TenCount + ends[i].XCount

			var arrows []ArrowScore
			_ = db.Select(&arrows, `
				SELECT arrow_number, score, is_x
				FROM qualification_arrow_scores
				WHERE end_score_uuid = ?
				ORDER BY arrow_number ASC
			`, ends[i].EndScoreUUID)
			if arrows == nil {
				arrows = []ArrowScore{}
			}
			ends[i].Arrows = arrows
		}

		c.JSON(http.StatusOK, MobileAssignmentScoreDetailResponse{
			Assignment: MobileAssignmentMeta{
				UUID:         meta.AssignmentUUID,
				SessionUUID:  meta.SessionUUID,
				SessionName:  meta.SessionName,
				EventName:    meta.EventName,
				TargetName:   meta.TargetName,
				ArcherName:   meta.ArcherName,
				Division:     meta.Division,
				AvatarURL:    meta.AvatarURL,
				TotalEnds:    meta.TotalEnds,
				ArrowsPerEnd: meta.ArrowsPerEnd,
			},
			Summary: MobileScoreSummary{
				TotalScore:    cumTotal,
				TotalX:        totalXCount,
				TotalTenPlusX: totalTenPlusXCount,
				EndsCompleted: len(ends),
			},
			Ends: castToEndScores(ends),
		})
	}
}

func castToEndScores(ends []EndScore) []MobileEndScore {
	res := make([]MobileEndScore, len(ends))
	for i, e := range ends {
		arrows := make([]MobileArrowScore, len(e.Arrows))
		for j, a := range e.Arrows {
			arrows[j] = MobileArrowScore(a)
		}
		res[i] = MobileEndScore{
			EndNumber:       e.EndNumber,
			EndScoreUUID:    e.EndScoreUUID,
			EndTotal:        e.EndTotal,
			XCount:          e.XCount,
			TenCount:        e.TenCount,
			CumulativeTotal: e.CumTotal,
			Arrows:          arrows,
		}
	}
	return res
}

func castToScannedArchersQual(archers []ArcherInfo) []MobileScannedArcherQualification {
	res := make([]MobileScannedArcherQualification, len(archers))
	for i, a := range archers {
		res[i] = MobileScannedArcherQualification(a)
	}
	return res
}

func castToScannedArchersElim(archers []ElimArcherInfo) []MobileScannedArcherElimination {
	res := make([]MobileScannedArcherElimination, len(archers))
	for i, a := range archers {
		res[i] = MobileScannedArcherElimination(a)
	}
	return res
}

// Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡ Archer Auth Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡

// MobileArcherLogin godoc
