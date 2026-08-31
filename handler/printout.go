package handler

import (
	"database/sql"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// ─────────────────────────────────────────────────────────────────────────────
// DATA STRUCTURES
// ─────────────────────────────────────────────────────────────────────────────

type PrintEventInfo struct {
	UUID      string         `db:"uuid"`
	Name      string         `db:"name"`
	Slug      string         `db:"slug"`
	Venue     sql.NullString `db:"venue"`
	Location  sql.NullString `db:"location"`
	City      sql.NullString `db:"city"`
	StartDate sql.NullTime   `db:"start_date"`
	EndDate   sql.NullTime   `db:"end_date"`
	LogoURL   sql.NullString `db:"logo_url"`
	OrgName   sql.NullString `db:"org_name"`
}

type PrintScoresheetEntry struct {
	ParticipantUUID string         `db:"participant_uuid"`
	TargetNo        int            `db:"target_no"`
	TargetName      string         `db:"target_name"`
	ArcherName      string         `db:"archer_name"`
	ArcherCode      sql.NullString `db:"archer_code"`
	ClubName        string         `db:"club_name"`
	CategoryName    string         `db:"category_name"`
	DivisionName    string         `db:"division_name"`
	AgeGroup        string         `db:"age_group"`
	Gender          string         `db:"gender"`
	Distance        sql.NullInt64  `db:"distance"`
	TargetFace      sql.NullString `db:"target_face"`
	SessionCode     string         `db:"session_code"`
	SessionName     string         `db:"session_name"`
	BirthDate       sql.NullTime   `db:"birth_date"`
	Email           sql.NullString `db:"email"`
}

type PrintParticipantRow struct {
	AthleteName     string         `db:"athlete_name"`
	AthleteCode     sql.NullString `db:"athlete_code"`
	ClubName        string         `db:"club_name"`
	AgeGroup        string         `db:"age_group"`
	DivisionName    string         `db:"division_name"`
	CategoryName    string         `db:"category_name"`
	SessionCode     string         `db:"session_code"`
	TargetName      string         `db:"target_name"`
	Gender          string         `db:"gender"`
	ParticipantUUID string         `db:"participant_uuid"`
}

type PrintClassStatRow struct {
	AgeGroup     string `db:"age_group"`
	DivisionName string `db:"division_name"`
	Gender       string `db:"gender"`
	TotalCount   int    `db:"total_count"`
}

type PrintClubStatRow struct {
	ClubName   string `db:"club_name"`
	TotalCount int    `db:"total_count"`
}

// ─────────────────────────────────────────────────────────────────────────────
// HELPER FUNCTIONS
// ─────────────────────────────────────────────────────────────────────────────

func fetchPrintEvent(db *sqlx.DB, eventID string) (*PrintEventInfo, error) {
	var ev PrintEventInfo
	query := `
		SELECT e.uuid, e.name, e.slug, e.venue, e.location, e.city, e.start_date, e.end_date, e.logo_url,
		       COALESCE(o.name, '') AS org_name
		FROM events e
		LEFT JOIN organizers o ON e.organizer_id = o.uuid
		WHERE e.uuid = ? OR e.slug = ?
		LIMIT 1
	`
	err := db.Get(&ev, query, eventID, eventID)
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

func formatPrintDateRange(start, end sql.NullTime) string {
	months := map[int]string{
		1: "Januari", 2: "Februari", 3: "Maret", 4: "April", 5: "Mei", 6: "Juni",
		7: "Juli", 8: "Agustus", 9: "September", 10: "Oktober", 11: "November", 12: "Desember",
	}
	if !start.Valid {
		return "-"
	}
	st := start.Time
	if !end.Valid || end.Time.Equal(st) {
		return fmt.Sprintf("%d %s %d", st.Day(), months[int(st.Month())], st.Year())
	}
	et := end.Time
	if st.Month() == et.Month() && st.Year() == et.Year() {
		return fmt.Sprintf("%d - %d %s %d", st.Day(), et.Day(), months[int(st.Month())], st.Year())
	}
	return fmt.Sprintf("%d %s %d - %d %s %d", st.Day(), months[int(st.Month())], st.Year(), et.Day(), months[int(et.Month())], et.Year())
}

func getPrintCommonCSS() string {
	return `
		@page {
			size: A4 portrait;
			margin: 8mm;
		}
		* {
			box-sizing: border-box;
			-webkit-print-color-adjust: exact !important;
			print-color-adjust: exact !important;
		}
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
			margin: 0;
			padding: 0;
			color: #0f172a;
			background: #ffffff;
			font-size: 11px;
			line-height: 1.3;
		}
		.page-break {
			page-break-after: always;
			break-after: page;
		}
		.avoid-break {
			page-break-inside: avoid;
			break-inside: avoid;
		}
		.no-print {
			display: block;
		}
		@media print {
			.no-print {
				display: none !important;
			}
			body {
				padding: 0 !important;
			}
		}
		.btn-print {
			position: fixed;
			bottom: 20px;
			right: 20px;
			background: #0f172a;
			color: #ffffff;
			padding: 12px 24px;
			border-radius: 12px;
			font-weight: 800;
			font-size: 13px;
			border: none;
			cursor: pointer;
			box-shadow: 0 4px 12px rgba(0,0,0,0.15);
			z-index: 9999;
			display: flex;
			align-items: center;
			gap: 8px;
		}
		.btn-print:hover {
			background: #1e293b;
		}
	`
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. QUALIFICATION SCORESHEET HANDLER (PURE GOLANG)
// ─────────────────────────────────────────────────────────────────────────────

func GetQualificationScoresheet(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		sessionCode := c.Param("sessionCode")

		categoryID := c.Query("category_id")
		targetFromStr := c.Query("target_from")
		targetToStr := c.Query("target_to")
		blankMode := c.Query("blank") == "1"
		autoPrint := c.Query("autoprint") == "1"
		showHeader := c.DefaultQuery("header", "1") == "1"
		showFlags := c.DefaultQuery("flags", "1") == "1"

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT
				COALESCE(ep.uuid, '') AS participant_uuid,
				COALESCE(et.board_number, 1) AS target_no,
				COALESCE(et.target_name, ep.target_name, '1A') AS target_name,
				COALESCE(a.full_name, 'Peserta Belum Ditentukan') AS archer_name,
				COALESCE(ep.back_number, CAST(a.id AS CHAR)) AS archer_code,
				COALESCE(cl.name, 'Individu / Tanpa Klub') AS club_name,
				COALESCE(CONCAT(rbt.name, ' ', rag.name, ' ', rgd.name), ec.category_name_custom, '-') AS category_name,
				COALESCE(rbt.name, '-') AS division_name,
				COALESCE(rag.name, '-') AS age_group,
				COALESCE(rgd.name, '-') AS gender,
				NULL as distance,
				NULL as target_face,
				COALESCE(qs.session_code, ?) AS session_code,
				COALESCE(qs.name, CONCAT('Sesi ', ?)) AS session_name,
				a.birth_date,
				a.email
			FROM event_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN qualification_target_assignments qta ON ep.uuid = qta.participant_uuid
			LEFT JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
			LEFT JOIN event_targets et ON qta.target_uuid = et.uuid
			WHERE ep.event_id = ?
		`

		var args []interface{}
		args = append(args, sessionCode, sessionCode, ev.UUID)

		if sessionCode != "" && sessionCode != "all" {
			query += " AND (qs.session_code = ? OR qs.session_code IS NULL)"
			args = append(args, sessionCode)
		}

		if categoryID != "" {
			query += " AND (ep.category_id = ? OR ec.uuid = ?)"
			args = append(args, categoryID, categoryID)
		}

		if targetFromStr != "" {
			if tFrom, err := strconv.Atoi(targetFromStr); err == nil {
				query += " AND et.board_number >= ?"
				args = append(args, tFrom)
			}
		}

		if targetToStr != "" {
			if tTo, err := strconv.Atoi(targetToStr); err == nil {
				query += " AND et.board_number <= ?"
				args = append(args, tTo)
			}
		}

		query += " ORDER BY et.board_number ASC, ep.target_name ASC, a.full_name ASC"

		var entries []PrintScoresheetEntry
		err = db.Select(&entries, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data scoresheet: " + err.Error()})
			return
		}

		if len(entries) == 0 && blankMode {
			entries = append(entries, PrintScoresheetEntry{
				TargetName:   "1A",
				ArcherName:   "........................................",
				ClubName:     "........................................",
				CategoryName: "........................................",
				SessionCode:  sessionCode,
				SessionName:  "Sesi " + sessionCode,
			})
		}

		dateStr := formatPrintDateRange(ev.StartDate, ev.EndDate)
		locationStr := html.EscapeString(ev.Venue.String)
		if ev.City.Valid && ev.City.String != "" {
			if locationStr != "" {
				locationStr += ", "
			}
			locationStr += html.EscapeString(ev.City.String)
		}

		var sb strings.Builder
		sb.WriteString("<!DOCTYPE html><html><head><meta charset='utf-8'>")
		sb.WriteString(fmt.Sprintf("<title>Scoresheet Kualifikasi - %s</title>", html.EscapeString(ev.Name)))
		sb.WriteString("<style>")
		sb.WriteString(getPrintCommonCSS())
		sb.WriteString(`
			.sheet-container {
				width: 100%;
				height: 138mm;
				border: 1.5px solid #0f172a;
				border-radius: 8px;
				padding: 5mm 6mm;
				margin-bottom: 6mm;
				background: #ffffff;
				display: flex;
				flex-direction: column;
				justify-content: space-between;
				position: relative;
				box-sizing: border-box;
			}
			.sheet-header {
				display: flex;
				justify-content: space-between;
				align-items: flex-start;
				border-bottom: 1.5px solid #0f172a;
				padding-bottom: 4px;
				margin-bottom: 4px;
			}
			.target-badge {
				font-size: 26px;
				font-weight: 900;
				color: #ffffff;
				background: #0f172a;
				padding: 2px 10px;
				border-radius: 6px;
				letter-spacing: 1px;
				display: inline-block;
			}
			.info-grid {
				display: grid;
				grid-template-columns: 1fr 1fr;
				gap: 4px;
				font-size: 10px;
				margin-bottom: 4px;
			}
			.info-item {
				display: flex;
				gap: 4px;
			}
			.info-label {
				font-weight: bold;
				color: #475569;
				min-width: 55px;
			}
			.info-value {
				font-weight: 900;
				color: #0f172a;
				text-transform: uppercase;
				overflow: hidden;
				text-overflow: ellipsis;
				white-space: nowrap;
			}
			table.scoresheet-table {
				width: 100%;
				border-collapse: collapse;
				font-size: 9.5px;
				text-align: center;
				margin-top: 2px;
			}
			table.scoresheet-table th {
				background: #f1f5f9;
				border: 1px solid #0f172a;
				padding: 3px 2px;
				font-weight: 900;
				font-size: 9px;
			}
			table.scoresheet-table td {
				border: 1px solid #0f172a;
				height: 15px;
				padding: 2px;
				font-weight: bold;
			}
			.sig-grid {
				display: grid;
				grid-template-columns: 1fr 1fr 1fr;
				gap: 10px;
				margin-top: 4px;
				padding-top: 4px;
				font-size: 9px;
				text-align: center;
			}
			.sig-line {
				border-bottom: 1px solid #0f172a;
				height: 18px;
				margin-bottom: 2px;
			}
		`)
		sb.WriteString("</style></head><body>")

		if autoPrint {
			sb.WriteString("<script>window.onload = function() { setTimeout(function(){ window.print(); }, 500); };</script>")
		}
		sb.WriteString("<button class='btn-print no-print' onclick='window.print()'>🖨️ Cetak Dokumen</button>")

		for i, entry := range entries {
			if i > 0 && i%2 == 0 {
				sb.WriteString("<div class='page-break'></div>")
			}

			sb.WriteString("<div class='sheet-container'>")

			// Header
			sb.WriteString("<div class='sheet-header'>")
			sb.WriteString("<div>")
			if showHeader {
				sb.WriteString(fmt.Sprintf("<div style='font-size: 13px; font-weight: 900; color: #0f172a;'>%s</div>", html.EscapeString(ev.Name)))
				sb.WriteString(fmt.Sprintf("<div style='font-size: 9px; color: #64748b;'>%s &bull; %s</div>", locationStr, dateStr))
			} else {
				sb.WriteString("<div style='font-size: 13px; font-weight: 900;'>LEMBAR SKOR RESMI KUALIFIKASI</div>")
			}
			sb.WriteString("</div>")

			sb.WriteString("<div style='text-align: right;'>")
			sb.WriteString(fmt.Sprintf("<div class='target-badge'>%s</div>", html.EscapeString(entry.TargetName)))
			sb.WriteString("</div>")
			sb.WriteString("</div>")

			// Athlete & Category Info
			sb.WriteString("<div class='info-grid'>")
			sb.WriteString("<div class='info-item'>")
			sb.WriteString("<span class='info-label'>Atlet:</span>")
			if blankMode {
				sb.WriteString("<span class='info-value'>..................................................</span>")
			} else {
				sb.WriteString(fmt.Sprintf("<span class='info-value'>%s</span>", html.EscapeString(entry.ArcherName)))
			}
			sb.WriteString("</div>")

			sb.WriteString("<div class='info-item'>")
			sb.WriteString("<span class='info-label'>Kategori:</span>")
			sb.WriteString(fmt.Sprintf("<span class='info-value'>%s</span>", html.EscapeString(entry.CategoryName)))
			sb.WriteString("</div>")

			if showFlags {
				sb.WriteString("<div class='info-item'>")
				sb.WriteString("<span class='info-label'>Klub:</span>")
				sb.WriteString(fmt.Sprintf("<span class='info-value'>%s</span>", html.EscapeString(entry.ClubName)))
				sb.WriteString("</div>")
			}

			sb.WriteString("<div class='info-item'>")
			sb.WriteString("<span class='info-label'>Sesi:</span>")
			sb.WriteString(fmt.Sprintf("<span class='info-value'>%s (%s)</span>", html.EscapeString(entry.SessionName), html.EscapeString(entry.SessionCode)))
			sb.WriteString("</div>")
			sb.WriteString("</div>")

			// Scoresheet Table (6 Ends, 6 Arrows Each)
			sb.WriteString("<table class='scoresheet-table'>")
			sb.WriteString("<thead><tr>")
			sb.WriteString("<th style='width: 30px;'>End</th>")
			sb.WriteString("<th style='width: 25px;'>1</th><th style='width: 25px;'>2</th><th style='width: 25px;'>3</th>")
			sb.WriteString("<th style='width: 25px;'>4</th><th style='width: 25px;'>5</th><th style='width: 25px;'>6</th>")
			sb.WriteString("<th style='width: 45px;'>End Total</th>")
			sb.WriteString("<th style='width: 55px;'>Running Total</th>")
			sb.WriteString("<th style='width: 30px;'>10+X</th><th style='width: 30px;'>X</th>")
			sb.WriteString("</tr></thead><tbody>")

			for endNum := 1; endNum <= 6; endNum++ {
				sb.WriteString("<tr>")
				sb.WriteString(fmt.Sprintf("<td style='background: #f8fafc; font-weight: 900;'>%d</td>", endNum))
				sb.WriteString("<td></td><td></td><td></td><td></td><td></td><td></td>")
				sb.WriteString("<td style='background: #f8fafc;'></td>")
				sb.WriteString("<td></td>")
				sb.WriteString("<td></td><td></td>")
				sb.WriteString("</tr>")
			}

			// Total Row
			sb.WriteString("<tr style='background: #f1f5f9; font-weight: 900;'>")
			sb.WriteString("<td colspan='7' style='text-align: right; padding-right: 8px;'>TOTAL SKOR BABAK INI</td>")
			sb.WriteString("<td style='font-size: 11px;'></td>")
			sb.WriteString("<td></td><td></td><td></td>")
			sb.WriteString("</tr>")

			sb.WriteString("</tbody></table>")

			// Signatures Footer
			sb.WriteString("<div class='sig-grid'>")
			sb.WriteString("<div><div class='sig-line'></div><strong>Tanda Tangan Atlet</strong></div>")
			sb.WriteString("<div><div class='sig-line'></div><strong>Pencatat Skor (Scorer)</strong></div>")
			sb.WriteString("<div><div class='sig-line'></div><strong>Wasit / Bantalan (Judge)</strong></div>")
			sb.WriteString("</div>")

			sb.WriteString("</div>")
		}

		sb.WriteString("</body></html>")

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, sb.String())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. PARTICIPANT LIST HANDLER (PURE GOLANG)
// ─────────────────────────────────────────────────────────────────────────────

func GetEventParticipantList(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		listType := c.DefaultQuery("type", "alphabetical")
		autoPrint := c.Query("autoprint") == "1"

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT
				a.full_name AS athlete_name,
				COALESCE(ep.back_number, CAST(a.id AS CHAR)) AS athlete_code,
				COALESCE(cl.name, 'Individu / Tanpa Klub') AS club_name,
				COALESCE(rag.name, '-') AS age_group,
				COALESCE(rbt.name, '-') AS division_name,
				COALESCE(rgd.name, '-') AS gender,
				COALESCE(CONCAT(rbt.name, ' ', rag.name, ' ', rgd.name), ec.category_name_custom, '-') AS category_name,
				COALESCE(qs.session_code, '-') AS session_code,
				COALESCE(et.target_name, ep.target_name, '-') AS target_name,
				ep.uuid AS participant_uuid
			FROM event_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN qualification_target_assignments qta ON ep.uuid = qta.participant_uuid
			LEFT JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
			LEFT JOIN event_targets et ON qta.target_uuid = et.uuid
			WHERE ep.event_id = ?
		`

		if listType == "by-club" {
			query += " ORDER BY cl.name ASC, a.full_name ASC"
		} else {
			query += " ORDER BY a.full_name ASC"
		}

		var participants []PrintParticipantRow
		err = db.Select(&participants, query, ev.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar peserta: " + err.Error()})
			return
		}

		docTitle := "Daftar Peserta (Urutan Abjad)"
		if listType == "by-club" {
			docTitle = "Daftar Peserta (Per Klub / Kontingen)"
		}

		dateStr := formatPrintDateRange(ev.StartDate, ev.EndDate)
		locationStr := html.EscapeString(ev.Venue.String)
		if ev.City.Valid && ev.City.String != "" {
			if locationStr != "" {
				locationStr += ", "
			}
			locationStr += html.EscapeString(ev.City.String)
		}

		var sb strings.Builder
		sb.WriteString("<!DOCTYPE html><html><head><meta charset='utf-8'>")
		sb.WriteString(fmt.Sprintf("<title>%s - %s</title>", docTitle, html.EscapeString(ev.Name)))
		sb.WriteString("<style>")
		sb.WriteString(getPrintCommonCSS())
		sb.WriteString(`
			.header-box {
				border-bottom: 2px solid #0f172a;
				padding-bottom: 8px;
				margin-bottom: 12px;
			}
			.group-title {
				background: #0f172a;
				color: #ffffff;
				font-size: 11px;
				font-weight: 900;
				padding: 4px 8px;
				margin: 12px 0 4px 0;
				border-radius: 4px;
			}
			table.report-table {
				width: 100%;
				border-collapse: collapse;
				font-size: 10px;
			}
			table.report-table th {
				background: #f1f5f9;
				border-bottom: 1.5px solid #0f172a;
				border-top: 1px solid #cbd5e1;
				padding: 5px 6px;
				font-weight: 900;
				text-align: left;
			}
			table.report-table td {
				border-bottom: 1px solid #e2e8f0;
				padding: 4px 6px;
			}
			table.report-table tr:nth-child(even) td {
				background: #f8fafc;
			}
			.grand-total {
				background: #0f172a;
				color: #ffffff;
				padding: 8px 12px;
				font-weight: 900;
				text-align: right;
				margin-top: 16px;
				border-radius: 6px;
				font-size: 11px;
			}
		`)
		sb.WriteString("</style></head><body>")

		if autoPrint {
			sb.WriteString("<script>window.onload = function() { setTimeout(function(){ window.print(); }, 500); };</script>")
		}
		sb.WriteString("<button class='btn-print no-print' onclick='window.print()'>🖨️ Cetak Dokumen</button>")

		// Header
		sb.WriteString("<div class='header-box'>")
		sb.WriteString(fmt.Sprintf("<div style='font-size: 16px; font-weight: 900; color: #0f172a;'>%s</div>", html.EscapeString(ev.Name)))
		sb.WriteString(fmt.Sprintf("<div style='font-size: 12px; font-weight: 800; color: #475569; margin: 2px 0;'>%s</div>", docTitle))
		sb.WriteString(fmt.Sprintf("<div style='font-size: 9px; color: #64748b;'>Lokasi: %s &bull; Tanggal: %s</div>", locationStr, dateStr))
		sb.WriteString("</div>")

		currentGroup := ""
		groupCount := 0
		rowNum := 0

		for _, p := range participants {
			groupKey := ""
			if listType == "by-club" {
				groupKey = p.ClubName
			} else {
				if len(p.AthleteName) > 0 {
					groupKey = strings.ToUpper(string([]rune(p.AthleteName)[0]))
				} else {
					groupKey = "#"
				}
			}

			if groupKey != currentGroup {
				if currentGroup != "" {
					sb.WriteString("</tbody></table>")
					sb.WriteString(fmt.Sprintf("<div style='text-align: right; font-size: 9px; font-weight: bold; color: #64748b; margin-bottom: 12px;'>Subtotal %s: %d Atlet</div>", html.EscapeString(currentGroup), groupCount))
				}

				currentGroup = groupKey
				groupCount = 0

				sb.WriteString(fmt.Sprintf("<div class='group-title'>%s</div>", html.EscapeString(currentGroup)))
				sb.WriteString("<table class='report-table'>")
				sb.WriteString("<thead><tr>")
				sb.WriteString("<th style='width: 30px;'>No</th>")
				sb.WriteString("<th style='width: 160px;'>Nama Atlet</th>")
				sb.WriteString("<th style='width: 140px;'>Klub / Kontingen</th>")
				sb.WriteString("<th style='width: 140px;'>Kategori Lomba</th>")
				sb.WriteString("<th style='width: 50px; text-align: center;'>Sesi</th>")
				sb.WriteString("<th style='width: 60px; text-align: center;'>Bantalan</th>")
				sb.WriteString("</tr></thead><tbody>")
			}

			rowNum++
			groupCount++

			sb.WriteString("<tr>")
			sb.WriteString(fmt.Sprintf("<td style='text-align: center; color: #64748b;'>%d</td>", rowNum))
			sb.WriteString(fmt.Sprintf("<td><strong>%s</strong></td>", html.EscapeString(p.AthleteName)))
			sb.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(p.ClubName)))
			sb.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(p.CategoryName)))
			sb.WriteString(fmt.Sprintf("<td style='text-align: center;'>%s</td>", html.EscapeString(p.SessionCode)))
			sb.WriteString(fmt.Sprintf("<td style='text-align: center; font-weight: 900;'>%s</td>", html.EscapeString(p.TargetName)))
			sb.WriteString("</tr>")
		}

		if currentGroup != "" {
			sb.WriteString("</tbody></table>")
			sb.WriteString(fmt.Sprintf("<div style='text-align: right; font-size: 9px; font-weight: bold; color: #64748b; margin-bottom: 12px;'>Subtotal %s: %d Atlet</div>", html.EscapeString(currentGroup), groupCount))
		}

		sb.WriteString(fmt.Sprintf("<div class='grand-total'>TOTAL KESELURUHAN PESERTA: %d ATLET</div>", len(participants)))
		sb.WriteString("</body></html>")

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, sb.String())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. ACCREDITATION BY SESSION HANDLER (PURE GOLANG)
// ─────────────────────────────────────────────────────────────────────────────

func GetEventAccreditationPrintout(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		autoPrint := c.Query("autoprint") == "1"

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT
				a.full_name AS athlete_name,
				COALESCE(ep.back_number, CAST(a.id AS CHAR)) AS athlete_code,
				COALESCE(cl.name, 'Individu / Tanpa Klub') AS club_name,
				COALESCE(rag.name, '-') AS age_group,
				COALESCE(rbt.name, '-') AS division_name,
				COALESCE(rgd.name, '-') AS gender,
				COALESCE(CONCAT(rbt.name, ' ', rag.name, ' ', rgd.name), ec.category_name_custom, '-') AS category_name,
				COALESCE(qs.session_code, '-') AS session_code,
				COALESCE(et.target_name, ep.target_name, '-') AS target_name,
				ep.uuid AS participant_uuid
			FROM event_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN qualification_target_assignments qta ON ep.uuid = qta.participant_uuid
			LEFT JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
			LEFT JOIN event_targets et ON qta.target_uuid = et.uuid
			WHERE ep.event_id = ?
			ORDER BY qs.session_code ASC, et.board_number ASC, ep.target_name ASC, a.full_name ASC
		`

		var participants []PrintParticipantRow
		err = db.Select(&participants, query, ev.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data akreditasi: " + err.Error()})
			return
		}

		dateStr := formatPrintDateRange(ev.StartDate, ev.EndDate)
		locationStr := html.EscapeString(ev.Venue.String)

		var sb strings.Builder
		sb.WriteString("<!DOCTYPE html><html><head><meta charset='utf-8'>")
		sb.WriteString(fmt.Sprintf("<title>Akreditasi Per Sesi - %s</title>", html.EscapeString(ev.Name)))
		sb.WriteString("<style>")
		sb.WriteString(getPrintCommonCSS())
		sb.WriteString(`
			.header-box {
				border-bottom: 2px solid #0f172a;
				padding-bottom: 8px;
				margin-bottom: 12px;
			}
			.session-header {
				background: #0f172a;
				color: #ffffff;
				padding: 6px 10px;
				font-size: 12px;
				font-weight: 900;
				border-radius: 4px;
				margin-top: 16px;
			}
			table.acc-table {
				width: 100%;
				border-collapse: collapse;
				font-size: 10px;
				margin-top: 4px;
			}
			table.acc-table th {
				background: #f1f5f9;
				border-bottom: 1.5px solid #0f172a;
				padding: 4px 6px;
				font-weight: 900;
				text-align: left;
			}
			table.acc-table td {
				border-bottom: 1px solid #e2e8f0;
				padding: 4px 6px;
			}
		`)
		sb.WriteString("</style></head><body>")

		if autoPrint {
			sb.WriteString("<script>window.onload = function() { setTimeout(function(){ window.print(); }, 500); };</script>")
		}
		sb.WriteString("<button class='btn-print no-print' onclick='window.print()'>🖨️ Cetak Dokumen</button>")

		// Header
		sb.WriteString("<div class='header-box'>")
		sb.WriteString(fmt.Sprintf("<div style='font-size: 16px; font-weight: 900; color: #0f172a;'>%s</div>", html.EscapeString(ev.Name)))
		sb.WriteString("<div style='font-size: 12px; font-weight: 800; color: #475569; margin: 2px 0;'>Daftar Akreditasi Peserta (Per Sesi & Bantalan)</div>")
		sb.WriteString(fmt.Sprintf("<div style='font-size: 9px; color: #64748b;'>Lokasi: %s &bull; Tanggal: %s</div>", locationStr, dateStr))
		sb.WriteString("</div>")

		currentSession := ""
		sessionCount := 0

		for _, p := range participants {
			if p.SessionCode != currentSession {
				if currentSession != "" {
					sb.WriteString("</tbody></table>")
					sb.WriteString(fmt.Sprintf("<div style='text-align: right; font-size: 9px; font-weight: bold; color: #64748b; margin-bottom: 12px;'>Total Sesi %s: %d Atlet</div>", html.EscapeString(currentSession), sessionCount))
					sb.WriteString("<div class='page-break'></div>")
				}

				currentSession = p.SessionCode
				sessionCount = 0

				sb.WriteString(fmt.Sprintf("<div class='session-header'>SESI %s</div>", html.EscapeString(currentSession)))
				sb.WriteString("<table class='acc-table'>")
				sb.WriteString("<thead><tr>")
				sb.WriteString("<th style='width: 70px; text-align: center;'>Bantalan</th>")
				sb.WriteString("<th style='width: 180px;'>Nama Atlet</th>")
				sb.WriteString("<th style='width: 150px;'>Klub / Kontingen</th>")
				sb.WriteString("<th style='width: 150px;'>Kategori</th>")
				sb.WriteString("<th style='width: 80px; text-align: center;'>Tanda Tangan</th>")
				sb.WriteString("</tr></thead><tbody>")
			}

			sessionCount++
			sb.WriteString("<tr>")
			sb.WriteString(fmt.Sprintf("<td style='text-align: center; font-weight: 900; font-size: 11px;'>%s</td>", html.EscapeString(p.TargetName)))
			sb.WriteString(fmt.Sprintf("<td><strong>%s</strong></td>", html.EscapeString(p.AthleteName)))
			sb.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(p.ClubName)))
			sb.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(p.CategoryName)))
			sb.WriteString("<td style='height: 20px; border-bottom: 1px dashed #cbd5e1;'></td>")
			sb.WriteString("</tr>")
		}

		if currentSession != "" {
			sb.WriteString("</tbody></table>")
			sb.WriteString(fmt.Sprintf("<div style='text-align: right; font-size: 9px; font-weight: bold; color: #64748b; margin-bottom: 12px;'>Total Sesi %s: %d Atlet</div>", html.EscapeString(currentSession), sessionCount))
		}

		sb.WriteString(fmt.Sprintf("<div class='grand-total' style='background: #0f172a; color: #ffffff; padding: 8px 12px; font-weight: 900; text-align: right; margin-top: 16px; border-radius: 6px;'>TOTAL KESELURUHAN PESERTA: %d ATLET</div>", len(participants)))
		sb.WriteString("</body></html>")

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, sb.String())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. PARTICIPANT STATISTICS (CLASSES & CLUBS) HANDLERS (PURE GOLANG)
// ─────────────────────────────────────────────────────────────────────────────

func GetEventStatisticsClasses(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		autoPrint := c.Query("autoprint") == "1"

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT
				COALESCE(rag.name, 'Umum') AS age_group,
				COALESCE(rbt.name, 'Standard') AS division_name,
				COALESCE(rgd.name, '-') AS gender,
				COUNT(ep.uuid) AS total_count
			FROM event_participants ep
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			WHERE ep.event_id = ?
			GROUP BY rag.name, rbt.name, rgd.name
			ORDER BY rag.name ASC, rbt.name ASC
		`

		var stats []PrintClassStatRow
		err = db.Select(&stats, query, ev.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil statistik kelas: " + err.Error()})
			return
		}

		var sb strings.Builder
		sb.WriteString("<!DOCTYPE html><html><head><meta charset='utf-8'>")
		sb.WriteString(fmt.Sprintf("<title>Statistik Kelas & Divisi - %s</title>", html.EscapeString(ev.Name)))
		sb.WriteString("<style>")
		sb.WriteString(getPrintCommonCSS())
		sb.WriteString(`
			.header-box { border-bottom: 2px solid #0f172a; padding-bottom: 8px; margin-bottom: 16px; }
			table.stat-table { width: 100%; border-collapse: collapse; font-size: 11px; }
			table.stat-table th { background: #f1f5f9; border: 1px solid #0f172a; padding: 6px 8px; font-weight: 900; }
			table.stat-table td { border: 1px solid #cbd5e1; padding: 6px 8px; }
		`)
		sb.WriteString("</style></head><body>")

		if autoPrint {
			sb.WriteString("<script>window.onload = function() { setTimeout(function(){ window.print(); }, 500); };</script>")
		}
		sb.WriteString("<button class='btn-print no-print' onclick='window.print()'>🖨️ Cetak Dokumen</button>")

		sb.WriteString("<div class='header-box'>")
		sb.WriteString(fmt.Sprintf("<div style='font-size: 16px; font-weight: 900;'>%s</div>", html.EscapeString(ev.Name)))
		sb.WriteString("<div style='font-size: 12px; font-weight: 800; color: #475569;'>Statistik Jumlah Peserta Berdasarkan Kelas & Divisi Lomba</div>")
		sb.WriteString("</div>")

		sb.WriteString("<table class='stat-table'>")
		sb.WriteString("<thead><tr>")
		sb.WriteString("<th style='width: 40px;'>No</th>")
		sb.WriteString("<th>Kelompok Usia</th>")
		sb.WriteString("<th>Divisi Busur</th>")
		sb.WriteString("<th>Kategori Gender</th>")
		sb.WriteString("<th style='width: 100px; text-align: right;'>Jumlah Atlet</th>")
		sb.WriteString("</tr></thead><tbody>")

		grandTotal := 0
		for i, s := range stats {
			grandTotal += s.TotalCount
			sb.WriteString("<tr>")
			sb.WriteString(fmt.Sprintf("<td style='text-align: center;'>%d</td>", i+1))
			sb.WriteString(fmt.Sprintf("<td><strong>%s</strong></td>", html.EscapeString(s.AgeGroup)))
			sb.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(s.DivisionName)))
			sb.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(s.Gender)))
			sb.WriteString(fmt.Sprintf("<td style='text-align: right; font-weight: 900;'>%d</td>", s.TotalCount))
			sb.WriteString("</tr>")
		}

		sb.WriteString("<tr style='background: #0f172a; color: #ffffff; font-weight: 900;'>")
		sb.WriteString("<td colspan='4' style='text-align: right;'>TOTAL KESELURUHAN PESERTA</td>")
		sb.WriteString(fmt.Sprintf("<td style='text-align: right; font-size: 12px;'>%d</td>", grandTotal))
		sb.WriteString("</tr>")
		sb.WriteString("</tbody></table></body></html>")

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, sb.String())
	}
}

func GetEventStatisticsClubs(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		autoPrint := c.Query("autoprint") == "1"

		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT
				COALESCE(cl.name, 'Individu / Tanpa Klub') AS club_name,
				COUNT(ep.uuid) AS total_count
			FROM event_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			WHERE ep.event_id = ?
			GROUP BY cl.name
			ORDER BY total_count DESC, cl.name ASC
		`

		var stats []PrintClubStatRow
		err = db.Select(&stats, query, ev.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil statistik klub: " + err.Error()})
			return
		}

		var sb strings.Builder
		sb.WriteString("<!DOCTYPE html><html><head><meta charset='utf-8'>")
		sb.WriteString(fmt.Sprintf("<title>Statistik Klub & Kontingen - %s</title>", html.EscapeString(ev.Name)))
		sb.WriteString("<style>")
		sb.WriteString(getPrintCommonCSS())
		sb.WriteString(`
			.header-box { border-bottom: 2px solid #0f172a; padding-bottom: 8px; margin-bottom: 16px; }
			table.stat-table { width: 100%; border-collapse: collapse; font-size: 11px; }
			table.stat-table th { background: #f1f5f9; border: 1px solid #0f172a; padding: 6px 8px; font-weight: 900; }
			table.stat-table td { border: 1px solid #cbd5e1; padding: 6px 8px; }
		`)
		sb.WriteString("</style></head><body>")

		if autoPrint {
			sb.WriteString("<script>window.onload = function() { setTimeout(function(){ window.print(); }, 500); };</script>")
		}
		sb.WriteString("<button class='btn-print no-print' onclick='window.print()'>🖨️ Cetak Dokumen</button>")

		sb.WriteString("<div class='header-box'>")
		sb.WriteString(fmt.Sprintf("<div style='font-size: 16px; font-weight: 900;'>%s</div>", html.EscapeString(ev.Name)))
		sb.WriteString("<div style='font-size: 12px; font-weight: 800; color: #475569;'>Statistik Kontribusi Peserta Berdasarkan Klub & Kontingen</div>")
		sb.WriteString("</div>")

		sb.WriteString("<table class='stat-table'>")
		sb.WriteString("<thead><tr>")
		sb.WriteString("<th style='width: 40px;'>No</th>")
		sb.WriteString("<th>Nama Klub / Kontingen</th>")
		sb.WriteString("<th style='width: 120px; text-align: right;'>Jumlah Peserta</th>")
		sb.WriteString("</tr></thead><tbody>")

		grandTotal := 0
		for i, s := range stats {
			grandTotal += s.TotalCount
			sb.WriteString("<tr>")
			sb.WriteString(fmt.Sprintf("<td style='text-align: center;'>%d</td>", i+1))
			sb.WriteString(fmt.Sprintf("<td><strong>%s</strong></td>", html.EscapeString(s.ClubName)))
			sb.WriteString(fmt.Sprintf("<td style='text-align: right; font-weight: 900;'>%d</td>", s.TotalCount))
			sb.WriteString("</tr>")
		}

		sb.WriteString("<tr style='background: #0f172a; color: #ffffff; font-weight: 900;'>")
		sb.WriteString("<td colspan='2' style='text-align: right;'>TOTAL KESELURUHAN PESERTA</td>")
		sb.WriteString(fmt.Sprintf("<td style='text-align: right; font-size: 12px;'>%d</td>", grandTotal))
		sb.WriteString("</tr>")
		sb.WriteString("</tbody></table></body></html>")

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, sb.String())
	}
}
