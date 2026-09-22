package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ─────────────────────────────────────────────────────────────────────────────
// DATA STRUCTURES
// ─────────────────────────────────────────────────────────────────────────────

type ScheduleItemDB struct {
	UUID            string         `db:"uuid" json:"uuid"`
	ScheduleUUID    sql.NullString `db:"schedule_uuid" json:"schedule_uuid"`
	TournamentID    string         `db:"tournament_id" json:"tournament_id"`
	ItemType        string         `db:"item_type" json:"item_type"` // 'general', 'qualification', 'elimination', 'finals'
	StartTime       string         `db:"start_time" json:"start_time"`
	EndTime         string         `db:"end_time" json:"end_time"`
	DurationMinutes int            `db:"duration_minutes" json:"duration_minutes"`
	DelayMinutes    int            `db:"delay_minutes" json:"delay_minutes"`
	Title           string         `db:"title" json:"title"`
	Subtitle        sql.NullString `db:"subtitle" json:"subtitle"`
	Description     sql.NullString `db:"description" json:"description"`
	Location        sql.NullString `db:"location" json:"location"`
	SessionCode     sql.NullString `db:"session_code" json:"session_code"`
	CategoryUUIDs   sql.NullString `db:"category_uuids" json:"category_uuids"`
	BracketUUID     sql.NullString `db:"bracket_uuid" json:"bracket_uuid"`
	ElimRound       sql.NullInt64  `db:"elim_round" json:"elim_round"`
	TargetStart     sql.NullInt64  `db:"target_start" json:"target_start"`
	TargetEnd       sql.NullInt64  `db:"target_end" json:"target_end"`
	SortOrder       int            `db:"sort_order" json:"sort_order"`
	DayNumber       int            `db:"day_number" json:"day_number"`
	ScheduleDate    sql.NullString `db:"schedule_date" json:"schedule_date"`
	CreatedAt       string         `db:"created_at" json:"created_at"`
	UpdatedAt       string         `db:"updated_at" json:"updated_at"`
}

type ScheduleItemResponse struct {
	UUID            string   `json:"uuid"`
	ScheduleUUID    string   `json:"schedule_uuid"`
	TournamentID    string   `json:"tournament_id"`
	ItemType        string   `json:"item_type"` // 'general', 'qualification', 'elimination', 'finals'
	StartTime       string   `json:"start_time"`
	EndTime         string   `json:"end_time"`
	DurationMinutes int      `json:"duration_minutes"`
	DelayMinutes    int      `json:"delay_minutes"`
	Title           string   `json:"title"`
	Subtitle        string   `json:"subtitle"`
	Description     string   `json:"description"`
	Location        string   `json:"location"`
	SessionCode     string   `json:"session_code"`
	CategoryUUIDs   []string `json:"category_uuids"`
	BracketUUID     string   `json:"bracket_uuid"`
	ElimRound       *int64   `json:"elim_round"`
	TargetStart     *int64   `json:"target_start"`
	TargetEnd       *int64   `json:"target_end"`
	SortOrder       int      `json:"sort_order"`
	DayNumber       int      `json:"day_number"`
	ScheduleDate    string   `json:"schedule_date"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

func (it ScheduleItemDB) ToResponse() ScheduleItemResponse {
	var catUUIDs []string
	if it.CategoryUUIDs.Valid && it.CategoryUUIDs.String != "" {
		_ = json.Unmarshal([]byte(it.CategoryUUIDs.String), &catUUIDs)
		if len(catUUIDs) == 0 && !strings.HasPrefix(it.CategoryUUIDs.String, "[") {
			catUUIDs = strings.Split(it.CategoryUUIDs.String, ",")
		}
	}
	if catUUIDs == nil {
		catUUIDs = []string{}
	}

	var elimRound *int64
	if it.ElimRound.Valid {
		v := it.ElimRound.Int64
		elimRound = &v
	}
	var targetStart *int64
	if it.TargetStart.Valid {
		v := it.TargetStart.Int64
		targetStart = &v
	}
	var targetEnd *int64
	if it.TargetEnd.Valid {
		v := it.TargetEnd.Int64
		targetEnd = &v
	}

	scheduleUUID := ""
	if it.ScheduleUUID.Valid {
		scheduleUUID = it.ScheduleUUID.String
	}
	subtitle := ""
	if it.Subtitle.Valid {
		subtitle = it.Subtitle.String
	}
	description := ""
	if it.Description.Valid {
		description = it.Description.String
	}
	location := ""
	if it.Location.Valid {
		location = it.Location.String
	}
	sessionCode := ""
	if it.SessionCode.Valid {
		sessionCode = it.SessionCode.String
	}
	bracketUUID := ""
	if it.BracketUUID.Valid {
		bracketUUID = it.BracketUUID.String
	}
	scheduleDate := ""
	if it.ScheduleDate.Valid {
		scheduleDate = it.ScheduleDate.String
	}

	return ScheduleItemResponse{
		UUID:            it.UUID,
		ScheduleUUID:    scheduleUUID,
		TournamentID:    it.TournamentID,
		ItemType:        it.ItemType,
		StartTime:       it.StartTime,
		EndTime:         it.EndTime,
		DurationMinutes: it.DurationMinutes,
		DelayMinutes:    it.DelayMinutes,
		Title:           it.Title,
		Subtitle:        subtitle,
		Description:     description,
		Location:        location,
		SessionCode:     sessionCode,
		CategoryUUIDs:   catUUIDs,
		BracketUUID:     bracketUUID,
		ElimRound:       elimRound,
		TargetStart:     targetStart,
		TargetEnd:       targetEnd,
		SortOrder:       it.SortOrder,
		DayNumber:       it.DayNumber,
		ScheduleDate:    scheduleDate,
		CreatedAt:       it.CreatedAt,
		UpdatedAt:       it.UpdatedAt,
	}
}

type ScheduleDayGroup struct {
	DayNumber    int                    `json:"day_number"`
	ScheduleDate string                 `json:"schedule_date"`
	DayLabel     string                 `json:"day_label"`
	Items        []ScheduleItemResponse `json:"items"`
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. GET FULL SCHEDULE TIMELINE
// ─────────────────────────────────────────────────────────────────────────────

func GetTournamentScheduleTimeline(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan"})
			return
		}

		var items []ScheduleItemDB
		query := `
			SELECT 
				uuid, schedule_uuid, tournament_id, item_type,
				CAST(start_time AS CHAR) AS start_time,
				CAST(end_time AS CHAR) AS end_time,
				duration_minutes, delay_minutes, title, subtitle, description, location,
				session_code, category_uuids, bracket_uuid, elim_round,
				target_start, target_end, sort_order, day_number,
				COALESCE(CAST(schedule_date AS CHAR), '') AS schedule_date,
				CAST(created_at AS CHAR) AS created_at,
				CAST(updated_at AS CHAR) AS updated_at
			FROM tournament_schedule_items
			WHERE tournament_id = ?
			ORDER BY COALESCE(NULLIF(schedule_date, ''), '9999-12-31') ASC, day_number ASC, start_time ASC, sort_order ASC
		`
		err = db.Select(&items, query, ev.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data jadwal", "details": err.Error()})
			return
		}

		// If no items in tournament_schedule_items, check if old tournament_schedules has data
		if len(items) == 0 {
			type OldScheduleRow struct {
				UUID        string         `db:"uuid"`
				Title       string         `db:"title"`
				Description sql.NullString `db:"description"`
				StartTime   sql.NullTime   `db:"start_time"`
				EndTime     sql.NullTime   `db:"end_time"`
				DayOrder    sql.NullInt64  `db:"day_order"`
				SortOrder   sql.NullInt64  `db:"sort_order"`
				Location    sql.NullString `db:"location"`
			}
			var oldRows []OldScheduleRow
			_ = db.Select(&oldRows, `
				SELECT uuid, title, description, start_time, end_time, day_order, sort_order, location
				FROM tournament_schedules
				WHERE tournament_id = ?
				ORDER BY COALESCE(day_order, 1) ASC, start_time ASC
			`, ev.UUID)

			for i, r := range oldRows {
				day := 1
				if r.DayOrder.Valid && r.DayOrder.Int64 > 0 {
					day = int(r.DayOrder.Int64)
				}
				st := "08:00:00"
				et := "09:00:00"
				dateStr := ""
				if r.StartTime.Valid {
					st = r.StartTime.Time.Format("15:04:05")
					dateStr = r.StartTime.Time.Format("2006-01-02")
				}
				if r.EndTime.Valid {
					et = r.EndTime.Time.Format("15:04:05")
				}

				item := ScheduleItemDB{
					UUID:            r.UUID,
					TournamentID:    ev.UUID,
					ItemType:        "general",
					StartTime:       st,
					EndTime:         et,
					DurationMinutes: 60,
					Title:           r.Title,
					Description:     r.Description,
					Location:        r.Location,
					SortOrder:       i + 1,
					DayNumber:       day,
					ScheduleDate:    sql.NullString{String: dateStr, Valid: dateStr != ""},
				}
				items = append(items, item)
			}
		}

		// Group items chronologically by date
		var groupOrder []string
		groupMap := make(map[string][]ScheduleItemResponse)

		for _, it := range items {
			dateKey := ""
			if it.ScheduleDate.Valid && it.ScheduleDate.String != "" {
				dateKey = it.ScheduleDate.String
				if len(dateKey) > 10 {
					dateKey = dateKey[:10]
				}
			} else if ev.StartDate.Valid {
				dateKey = ev.StartDate.Time.AddDate(0, 0, it.DayNumber-1).Format("2006-01-02")
			}
			if dateKey == "" {
				dateKey = fmt.Sprintf("no-date-%d", it.DayNumber)
			}

			if _, exists := groupMap[dateKey]; !exists {
				groupOrder = append(groupOrder, dateKey)
			}
			groupMap[dateKey] = append(groupMap[dateKey], it.ToResponse())
		}

		// Sort groupOrder chronologically
		sort.SliceStable(groupOrder, func(i, j int) bool {
			return groupOrder[i] < groupOrder[j]
		})

		var days []ScheduleDayGroup
		for idx, key := range groupOrder {
			dayNum := idx + 1
			dayItems := groupMap[key]

			dateLabel := fmt.Sprintf("Hari %d", dayNum)
			dateVal := ""
			t, pErr := time.Parse("2006-01-02", key)
			if pErr == nil {
				dateVal = key
				dateLabel = t.Format("Monday, 02 Jan 2006")
			} else if len(dayItems) > 0 && dayItems[0].ScheduleDate != "" {
				dateVal = dayItems[0].ScheduleDate
				if t2, pErr2 := time.Parse("2006-01-02", dateVal); pErr2 == nil {
					dateLabel = t2.Format("Monday, 02 Jan 2006")
				}
			}

			// Sync day_number in items response
			for i := range dayItems {
				dayItems[i].DayNumber = dayNum
				if dateVal != "" {
					dayItems[i].ScheduleDate = dateVal
				}
			}

			days = append(days, ScheduleDayGroup{
				DayNumber:    dayNum,
				ScheduleDate: dateVal,
				DayLabel:     dateLabel,
				Items:        dayItems,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"tournament_id": ev.UUID,
			"days":          days,
			"total_items":   len(items),
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. SAVE (CREATE / UPDATE) SCHEDULE ITEM
// ─────────────────────────────────────────────────────────────────────────────

type SaveScheduleItemRequest struct {
	UUID            string   `json:"uuid"`
	DayNumber       int      `json:"day_number"`
	ScheduleDate    string   `json:"schedule_date"`
	ItemType        string   `json:"item_type" binding:"required"` // general, qualification, elimination, finals, break
	StartTime       string   `json:"start_time" binding:"required"`
	EndTime         string   `json:"end_time" binding:"required"`
	DurationMinutes int      `json:"duration_minutes"`
	Title           string   `json:"title" binding:"required"`
	Subtitle        string   `json:"subtitle"`
	Description     string   `json:"description"`
	Location        string   `json:"location"`
	SessionCode     string   `json:"session_code"`
	CategoryUUIDs   []string `json:"category_uuids"`
	BracketUUID     string   `json:"bracket_uuid"`
	ElimRound       *int     `json:"elim_round"`
	TargetStart     *int     `json:"target_start"`
	TargetEnd       *int     `json:"target_end"`
	SortOrder       int      `json:"sort_order"`
}

func SaveTournamentScheduleItem(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan"})
			return
		}

		var req SaveScheduleItemRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data permohonan tidak valid", "details": err.Error()})
			return
		}

		// Format time string
		st := strings.TrimSpace(req.StartTime)
		if len(st) == 5 {
			st += ":00"
		}
		et := strings.TrimSpace(req.EndTime)
		if len(et) == 5 {
			et += ":00"
		}

		// Validate time format and order
		t1, err1 := time.Parse("15:04:05", st)
		t2, err2 := time.Parse("15:04:05", et)
		if err1 != nil || err2 != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format waktu tidak valid (Gunakan format HH:MM)"})
			return
		}
		if !t2.After(t1) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Waktu selesai harus lebih besar dari waktu mulai"})
			return
		}

		// Type-specific smart validation
		if req.ItemType == "elimination" || req.ItemType == "finals" {
			if req.ElimRound == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Babak eliminasi wajib dipilih untuk sesi Eliminasi / Final"})
				return
			}
		}

		// Validate target range if provided
		if req.TargetStart != nil && req.TargetEnd != nil {
			if *req.TargetStart < 1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Target awal minimal nomor 1"})
				return
			}
			if *req.TargetEnd < *req.TargetStart {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Target akhir tidak boleh lebih kecil dari target awal"})
				return
			}
		} else if req.TargetStart != nil && req.TargetEnd == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Silakan lengkapi juga target akhir"})
			return
		} else if req.TargetStart == nil && req.TargetEnd != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Silakan lengkapi juga target awal"})
			return
		}

		// Calculate duration
		dur := req.DurationMinutes
		if dur <= 0 {
			dur = int(t2.Sub(t1).Minutes())
			if dur <= 0 {
				dur = 30
			}
		}

		// Auto-derive DayNumber if not explicitly set
		dayNum := req.DayNumber
		if dayNum <= 0 {
			if req.ScheduleDate != "" && ev.StartDate.Valid {
				parsedDate, pErr := time.Parse("2006-01-02", req.ScheduleDate)
				if pErr == nil {
					diff := int(parsedDate.Sub(ev.StartDate.Time).Hours()/24) + 1
					if diff >= 1 {
						dayNum = diff
					}
				}
			}
			if dayNum <= 0 {
				dayNum = 1
			}
		}

		itemUUID := strings.TrimSpace(req.UUID)
		if itemUUID == "" {
			itemUUID = uuid.New().String()
		}

		// Check for exact duplicate (same tournament, same date/day, same start/end time, same title)
		var duplicateCount int
		dateCheckVal := req.ScheduleDate
		if len(dateCheckVal) > 10 {
			dateCheckVal = dateCheckVal[:10]
		}
		checkQuery := `
			SELECT COUNT(*) 
			FROM tournament_schedule_items 
			WHERE tournament_id = ? 
			  AND (
				(? != '' AND (schedule_date = ? OR schedule_date LIKE CONCAT(?, '%'))) OR
				(? = '' AND day_number = ?)
			  )
			  AND TIME_FORMAT(start_time, '%H:%i') = TIME_FORMAT(?, '%H:%i')
			  AND TIME_FORMAT(end_time, '%H:%i') = TIME_FORMAT(?, '%H:%i')
			  AND LOWER(TRIM(title)) = LOWER(TRIM(?))
			  AND uuid != ?
		`
		_ = db.Get(&duplicateCount, checkQuery, ev.UUID, dateCheckVal, dateCheckVal, dateCheckVal, dateCheckVal, dayNum, st, et, req.Title, itemUUID)
		if duplicateCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Jadwal dengan judul dan rentang waktu tersebut sudah terdaftar pada hari ini.",
				"code":  "duplicate_schedule",
			})
			return
		}

		catJSON := "[]"
		if len(req.CategoryUUIDs) > 0 {
			b, _ := json.Marshal(req.CategoryUUIDs)
			catJSON = string(b)
		}

		// Upsert query
		query := `
			INSERT INTO tournament_schedule_items (
				uuid, tournament_id, item_type, start_time, end_time,
				duration_minutes, delay_minutes, title, subtitle, description,
				location, session_code, category_uuids, bracket_uuid, elim_round,
				target_start, target_end, sort_order, day_number, schedule_date
			) VALUES (
				?, ?, ?, ?, ?,
				?, 0, ?, ?, ?,
				?, ?, ?, ?, ?,
				?, ?, ?, ?, ?
			)
			ON DUPLICATE KEY UPDATE
				item_type = VALUES(item_type),
				start_time = VALUES(start_time),
				end_time = VALUES(end_time),
				duration_minutes = VALUES(duration_minutes),
				title = VALUES(title),
				subtitle = VALUES(subtitle),
				description = VALUES(description),
				location = VALUES(location),
				session_code = VALUES(session_code),
				category_uuids = VALUES(category_uuids),
				bracket_uuid = VALUES(bracket_uuid),
				elim_round = VALUES(elim_round),
				target_start = VALUES(target_start),
				target_end = VALUES(target_end),
				sort_order = VALUES(sort_order),
				day_number = VALUES(day_number),
				schedule_date = VALUES(schedule_date),
				updated_at = CURRENT_TIMESTAMP
		`

		var schedDate interface{} = nil
		if req.ScheduleDate != "" {
			sDate := strings.TrimSpace(req.ScheduleDate)
			if len(sDate) >= 10 {
				schedDate = sDate[:10]
			} else {
				schedDate = sDate
			}
		} else if ev.StartDate.Valid {
			schedDate = ev.StartDate.Time.AddDate(0, 0, dayNum-1).Format("2006-01-02")
		}

		_, err = db.Exec(
			query,
			itemUUID, ev.UUID, req.ItemType, st, et,
			dur, req.Title, req.Subtitle, req.Description,
			req.Location, req.SessionCode, catJSON, req.BracketUUID, req.ElimRound,
			req.TargetStart, req.TargetEnd, req.SortOrder, dayNum, schedDate,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan item jadwal", "details": err.Error()})
			return
		}

		// Reindex day numbers to ensure 100% chronological consistency
		reindexTournamentScheduleDays(db, ev.UUID)

		c.JSON(http.StatusOK, gin.H{
			"message": "Item jadwal berhasil disimpan",
			"uuid":    itemUUID,
		})
	}
}

// Helper to reindex day_number chronologically for all items of a tournament
func reindexTournamentScheduleDays(db *sqlx.DB, tournamentUUID string) {
	type DateRow struct {
		ScheduleDate string `db:"schedule_date"`
	}
	var dates []DateRow
	err := db.Select(&dates, `
		SELECT DISTINCT COALESCE(CAST(schedule_date AS CHAR), '') AS schedule_date
		FROM tournament_schedule_items
		WHERE tournament_id = ?
		ORDER BY COALESCE(NULLIF(schedule_date, ''), '9999-12-31') ASC
	`, tournamentUUID)
	if err != nil {
		return
	}

	for idx, d := range dates {
		dayNum := idx + 1
		if d.ScheduleDate != "" {
			_, _ = db.Exec(`
				UPDATE tournament_schedule_items
				SET day_number = ?
				WHERE tournament_id = ? AND schedule_date = ?
			`, dayNum, tournamentUUID, d.ScheduleDate)
		} else {
			_, _ = db.Exec(`
				UPDATE tournament_schedule_items
				SET day_number = ?
				WHERE tournament_id = ? AND (schedule_date IS NULL OR schedule_date = '')
			`, dayNum, tournamentUUID)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. DELETE SCHEDULE ITEM
// ─────────────────────────────────────────────────────────────────────────────

func DeleteTournamentScheduleItem(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		itemID := c.Param("itemId")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan"})
			return
		}

		_, err = db.Exec(`DELETE FROM tournament_schedule_items WHERE uuid = ? AND tournament_id = ?`, itemID, ev.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus item jadwal", "details": err.Error()})
			return
		}

		// Also cleanup from legacy tournament_schedules if present
		_, _ = db.Exec(`DELETE FROM tournament_schedules WHERE uuid = ? AND tournament_id = ?`, itemID, ev.UUID)

		// Reindex day numbers to ensure 100% chronological consistency
		reindexTournamentScheduleDays(db, ev.UUID)

		c.JSON(http.StatusOK, gin.H{"message": "Item jadwal berhasil dihapus"})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. AUTO-GENERATE SCHEDULE FROM QUALIFICATION SESSIONS & ELIMINATION BRACKETS
// ─────────────────────────────────────────────────────────────────────────────

func AutoGenerateTournamentSchedule(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan"})
			return
		}

		// 1. Fetch Qualification Sessions
		type QualSessionData struct {
			UUID        string         `db:"uuid"`
			SessionCode string         `db:"session_code"`
			Name        string         `db:"name"`
			SessionDate sql.NullTime   `db:"session_date"`
			StartTime   sql.NullTime   `db:"start_time"`
			EndTime     sql.NullTime   `db:"end_time"`
		}
		var sessions []QualSessionData
		_ = db.Select(&sessions, `
			SELECT uuid, session_code, name, session_date, start_time, end_time
			FROM qualification_sessions
			WHERE tournament_uuid = ?
			ORDER BY session_date ASC, start_time ASC
		`, ev.UUID)

		// 2. Fetch Categories
		type CatData struct {
			UUID         string `db:"uuid"`
			CategoryName string `db:"category_name"`
		}
		var categories []CatData
		_ = db.Select(&categories, `
			SELECT tc.uuid, COALESCE(CONCAT(rbt.name, ' ', rag.name, ' ', rgd.name), tc.category_name_custom, 'Kategori') AS category_name
			FROM tournament_categories tc
			LEFT JOIN ref_bow_types rbt ON tc.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON tc.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON tc.gender_division_uuid = rgd.uuid
			WHERE tc.tournament_id = ?
		`, ev.UUID)

		// Clear existing generated items for this tournament
		_, _ = db.Exec(`DELETE FROM tournament_schedule_items WHERE tournament_id = ?`, ev.UUID)

		var generatedItems []SaveScheduleItemRequest

		// Day 1: Official Practice & Qualification
		day1Date := ""
		if ev.StartDate.Valid {
			day1Date = ev.StartDate.Time.Format("2006-01-02")
		}

		// Agenda 1: TM & Official Practice
		generatedItems = append(generatedItems, SaveScheduleItemRequest{
			UUID:            uuid.New().String(),
			DayNumber:       1,
			ScheduleDate:    day1Date,
			ItemType:        "general",
			StartTime:       "07:30:00",
			EndTime:         "08:30:00",
			DurationMinutes: 60,
			Title:           "Official Practice & Equipment Inspection",
			Subtitle:        "All Categories & Divisions",
			Location:        "Main Field",
			SortOrder:       1,
		})

		// Agenda 2: Sesi Kualifikasi
		if len(sessions) > 0 {
			for idx, ses := range sessions {
				st := "08:30:00"
				et := "11:30:00"
				if idx == 1 {
					st = "13:00:00"
					et = "16:00:00"
				}
				if ses.StartTime.Valid {
					st = ses.StartTime.Time.Format("15:04:05")
				}
				if ses.EndTime.Valid {
					et = ses.EndTime.Time.Format("15:04:05")
				}

				t1, _ := time.Parse("15:04:05", st)
				t2, _ := time.Parse("15:04:05", et)
				dur := int(t2.Sub(t1).Minutes())
				if dur <= 0 {
					dur = 180
				}

				generatedItems = append(generatedItems, SaveScheduleItemRequest{
					UUID:            uuid.New().String(),
					DayNumber:       1,
					ScheduleDate:    day1Date,
					ItemType:        "qualification",
					StartTime:       st,
					EndTime:         et,
					DurationMinutes: dur,
					Title:           fmt.Sprintf("Qualification Round - %s", ses.Name),
					Subtitle:        "Scoring 72 Arrows (2 x 36)",
					SessionCode:     ses.SessionCode,
					Location:        "Main Field",
					TargetStart:     intPtr(1),
					TargetEnd:       intPtr(32),
					SortOrder:       idx + 2,
				})
			}
		} else {
			// Fallback qualification item
			generatedItems = append(generatedItems, SaveScheduleItemRequest{
				UUID:            uuid.New().String(),
				DayNumber:       1,
				ScheduleDate:    day1Date,
				ItemType:        "qualification",
				StartTime:       "08:30:00",
				EndTime:         "12:00:00",
				DurationMinutes: 210,
				Title:           "Qualification Round - Session 1",
				Subtitle:        "Recurve, Compound & Barebow Divisions",
				Location:        "Main Field",
				TargetStart:     intPtr(1),
				TargetEnd:       intPtr(30),
				SortOrder:       2,
			})
		}

		// Day 2: Individual Elimination Matches
		day2Date := ""
		if ev.StartDate.Valid {
			day2Date = ev.StartDate.Time.AddDate(0, 0, 1).Format("2006-01-02")
		}

		generatedItems = append(generatedItems, SaveScheduleItemRequest{
			UUID:            uuid.New().String(),
			DayNumber:       2,
			ScheduleDate:    day2Date,
			ItemType:        "general",
			StartTime:       "08:00:00",
			EndTime:         "08:30:00",
			DurationMinutes: 30,
			Title:           "Warm-up / Practice Ends",
			Subtitle:        "Individual Finalists",
			Location:        "Main Field",
			SortOrder:       1,
		})

		generatedItems = append(generatedItems, SaveScheduleItemRequest{
			UUID:            uuid.New().String(),
			DayNumber:       2,
			ScheduleDate:    day2Date,
			ItemType:        "elimination",
			StartTime:       "08:30:00",
			EndTime:         "09:15:00",
			DurationMinutes: 45,
			Title:           "1/16 Elimination Matches",
			Subtitle:        "Individual Recurve & Compound",
			Location:        "Main Field",
			ElimRound:       intPtr(16),
			TargetStart:     intPtr(1),
			TargetEnd:       intPtr(32),
			SortOrder:       2,
		})

		generatedItems = append(generatedItems, SaveScheduleItemRequest{
			UUID:            uuid.New().String(),
			DayNumber:       2,
			ScheduleDate:    day2Date,
			ItemType:        "elimination",
			StartTime:       "09:30:00",
			EndTime:         "10:15:00",
			DurationMinutes: 45,
			Title:           "1/8 Elimination Matches",
			Subtitle:        "Individual All Categories",
			Location:        "Main Field",
			ElimRound:       intPtr(8),
			TargetStart:     intPtr(1),
			TargetEnd:       intPtr(16),
			SortOrder:       3,
		})

		generatedItems = append(generatedItems, SaveScheduleItemRequest{
			UUID:            uuid.New().String(),
			DayNumber:       2,
			ScheduleDate:    day2Date,
			ItemType:        "elimination",
			StartTime:       "10:30:00",
			EndTime:         "11:15:00",
			DurationMinutes: 45,
			Title:           "Quarterfinals (1/4)",
			Subtitle:        "Individual All Categories",
			Location:        "Main Field",
			ElimRound:       intPtr(4),
			TargetStart:     intPtr(1),
			TargetEnd:       intPtr(8),
			SortOrder:       4,
		})

		generatedItems = append(generatedItems, SaveScheduleItemRequest{
			UUID:            uuid.New().String(),
			DayNumber:       2,
			ScheduleDate:    day2Date,
			ItemType:        "elimination",
			StartTime:       "11:30:00",
			EndTime:         "12:15:00",
			DurationMinutes: 45,
			Title:           "Semifinals (1/2)",
			Subtitle:        "Individual All Categories",
			Location:        "Main Field",
			ElimRound:       intPtr(2),
			TargetStart:     intPtr(1),
			TargetEnd:       intPtr(4),
			SortOrder:       5,
		})

		// Day 3: Team Matches, Finals & Medal Ceremony
		day3Date := ""
		if ev.StartDate.Valid {
			day3Date = ev.StartDate.Time.AddDate(0, 0, 2).Format("2006-01-02")
		}

		generatedItems = append(generatedItems, SaveScheduleItemRequest{
			UUID:            uuid.New().String(),
			DayNumber:       3,
			ScheduleDate:    day3Date,
			ItemType:        "elimination",
			StartTime:       "08:30:00",
			EndTime:         "10:30:00",
			DurationMinutes: 120,
			Title:           "Team Elimination Matches (Quarterfinals to Finals)",
			Subtitle:        "Men, Women & Mixed Team Divisions",
			Location:        "Main Field",
			TargetStart:     intPtr(1),
			TargetEnd:       intPtr(12),
			SortOrder:       1,
		})

		generatedItems = append(generatedItems, SaveScheduleItemRequest{
			UUID:            uuid.New().String(),
			DayNumber:       3,
			ScheduleDate:    day3Date,
			ItemType:        "finals",
			StartTime:       "10:45:00",
			EndTime:         "12:30:00",
			DurationMinutes: 105,
			Title:           "Bronze & Gold Medal Matches",
			Subtitle:        "Individual Finals (Live Arena Matches)",
			Location:        "Final Field / Arena",
			ElimRound:       intPtr(1),
			TargetStart:     intPtr(1),
			TargetEnd:       intPtr(2),
			SortOrder:       2,
		})

		generatedItems = append(generatedItems, SaveScheduleItemRequest{
			UUID:            uuid.New().String(),
			DayNumber:       3,
			ScheduleDate:    day3Date,
			ItemType:        "general",
			StartTime:       "13:30:00",
			EndTime:         "15:00:00",
			DurationMinutes: 90,
			Title:           "Award Ceremony & Closing",
			Subtitle:        "Penyerahan Medali & Piagam Penghargaan",
			Location:        "Podium Area",
			SortOrder:       3,
		})

		// Insert all generated items
		for _, it := range generatedItems {
			query := `
				INSERT INTO tournament_schedule_items (
					uuid, tournament_id, item_type, start_time, end_time,
					duration_minutes, delay_minutes, title, subtitle, description,
					location, session_code, category_uuids, bracket_uuid, elim_round,
					target_start, target_end, sort_order, day_number, schedule_date
				) VALUES (
					?, ?, ?, ?, ?,
					?, 0, ?, ?, ?,
					?, ?, '[]', ?, ?,
					?, ?, ?, ?, ?
				)
			`
			_, _ = db.Exec(
				query,
				it.UUID, ev.UUID, it.ItemType, it.StartTime, it.EndTime,
				it.DurationMinutes, it.Title, it.Subtitle, it.Description,
				it.Location, it.SessionCode, it.BracketUUID, it.ElimRound,
				it.TargetStart, it.TargetEnd, it.SortOrder, it.DayNumber, it.ScheduleDate,
			)
		}

		c.JSON(http.StatusOK, gin.H{
			"message":     "Jadwal turnamen berhasil dibuat secara otomatis",
			"total_items": len(generatedItems),
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 5. SHIFT SCHEDULE DELAY
// ─────────────────────────────────────────────────────────────────────────────

type ShiftDelayRequest struct {
	DayNumber    int `json:"day_number" binding:"required"`
	ShiftMinutes int `json:"shift_minutes" binding:"required"` // e.g. +15, +30, -15
	FromTime     string `json:"from_time"`                      // e.g. '09:00:00'
}

func ShiftTournamentScheduleDelay(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan"})
			return
		}

		var req ShiftDelayRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data permohonan shift tidak valid", "details": err.Error()})
			return
		}

		var items []ScheduleItemDB
		query := `
			SELECT uuid, start_time, end_time, duration_minutes, delay_minutes
			FROM tournament_schedule_items
			WHERE tournament_id = ? AND day_number = ?
		`
		args := []interface{}{ev.UUID, req.DayNumber}
		if req.FromTime != "" {
			query += " AND start_time >= ?"
			args = append(args, req.FromTime)
		}
		query += " ORDER BY start_time ASC"

		_ = db.Select(&items, query, args...)

		for _, it := range items {
			st, err1 := time.Parse("15:04:05", it.StartTime)
			et, err2 := time.Parse("15:04:05", it.EndTime)
			if err1 == nil && err2 == nil {
				newST := st.Add(time.Duration(req.ShiftMinutes) * time.Minute).Format("15:04:05")
				newET := et.Add(time.Duration(req.ShiftMinutes) * time.Minute).Format("15:04:05")
				newDelay := it.DelayMinutes + req.ShiftMinutes

				_, _ = db.Exec(`
					UPDATE tournament_schedule_items 
					SET start_time = ?, end_time = ?, delay_minutes = ?
					WHERE uuid = ?
				`, newST, newET, newDelay, it.UUID)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message":       fmt.Sprintf("Jadwal hari ke-%d berhasil digeser sebanyak %+d menit", req.DayNumber, req.ShiftMinutes),
			"shifted_items": len(items),
		})
	}
}

func intPtr(i int) *int {
	return &i
}
