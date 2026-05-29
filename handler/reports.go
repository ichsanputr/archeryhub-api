package handler

import (
	"fmt"
	"net/http"

	"Archeris-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// splitcount represents a label and count pair for splits
type splitcount struct {
	Label string `db:"label" json:"name"`
	Count int    `db:"count" json:"count"`
}

// splitamount represents a label, transaction count, and amount sum
type splitamount struct {
	Label  string  `db:"label" json:"name"`
	Count  int     `db:"count" json:"count"`
	Amount float64 `db:"amount" json:"amount"`
}

// trendpoint represents a date and count/amount pair for trendlines
type trendpoint struct {
	Label string  `db:"label" json:"date"`
	Value float64 `db:"value" json:"value"`
}

// eventoption represents a minimal event representation for filters
type eventoption struct {
	UUID      string  `db:"uuid" json:"id"`
	Name      string  `db:"name" json:"name"`
	StartDate *string `db:"start_date" json:"start_date"`
	EndDate   *string `db:"end_date" json:"end_date"`
}

// getparticipantsreport returns statistics and recent list of participants
func GetOrganizationParticipantsReport(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "tidak diizinkan"})
			return
		}

		orgID := userID.(string)

		// build queries
		whereClause := "e.organizer_id = ?"
		args := []interface{}{orgID}

		if eventID := c.Query("event_id"); eventID != "" && eventID != "all" {
			whereClause += " AND e.uuid = ?"
			args = append(args, eventID)
		}

		if startDate := c.Query("start_date"); startDate != "" {
			whereClause += " AND ep.registration_date >= ?"
			args = append(args, startDate+" 00:00:00")
		}

		if endDate := c.Query("end_date"); endDate != "" {
			whereClause += " AND ep.registration_date <= ?"
			args = append(args, endDate+" 23:59:59")
		}

		// additional optional filters
		if gender := c.Query("gender"); gender != "" && gender != "all" {
			whereClause += " AND rgd.name = ?"
			args = append(args, gender)
		}

		if bowType := c.Query("bow_type"); bowType != "" && bowType != "all" {
			whereClause += " AND rbt.name = ?"
			args = append(args, bowType)
		}

		if status := c.Query("status"); status != "" && status != "all" {
			if status == "checked_in" {
				whereClause += " AND ep.last_reregistration_at IS NOT NULL"
			} else if status == "pending" {
				whereClause += " AND ep.last_reregistration_at IS NULL"
			}
		}

		// 1. fetch total participants count
		var totalParticipants int
		err := db.Get(&totalParticipants, fmt.Sprintf(`
			SELECT COUNT(*) 
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			WHERE %s
		`, whereClause), args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal mengambil jumlah peserta", "details": err.Error()})
			return
		}

		// 2. gender split
		var genderSplit []splitcount
		_ = db.Select(&genderSplit, fmt.Sprintf(`
			SELECT COALESCE(rgd.name, 'tidak diketahui') as label, COUNT(*) as count
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			WHERE %s
			GROUP BY label
		`, whereClause), args...)

		// 3. bow type split
		var bowTypeSplit []splitcount
		_ = db.Select(&bowTypeSplit, fmt.Sprintf(`
			SELECT COALESCE(rbt.name, 'tidak diketahui') as label, COUNT(*) as count
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			WHERE %s
			GROUP BY label
		`, whereClause), args...)

		// 4. age group split
		var ageGroupSplit []splitcount
		_ = db.Select(&ageGroupSplit, fmt.Sprintf(`
			SELECT COALESCE(rag.name, 'tidak diketahui') as label, COUNT(*) as count
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			WHERE %s
			GROUP BY label
		`, whereClause), args...)

		// 5. registration source split
		var sourceSplit []splitcount
		_ = db.Select(&sourceSplit, fmt.Sprintf(`
			SELECT COALESCE(ep.registration_source, 'self_register') as label, COUNT(*) as count
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			WHERE %s
			GROUP BY label
		`, whereClause), args...)

		// 6. check-in status
		var reregStats struct {
			CheckedIn int `db:"checked_in"`
			Pending   int `db:"pending"`
		}
		_ = db.Get(&reregStats, fmt.Sprintf(`
			SELECT 
				COALESCE(SUM(CASE WHEN ep.last_reregistration_at IS NOT NULL THEN 1 ELSE 0 END), 0) as checked_in,
				COALESCE(SUM(CASE WHEN ep.last_reregistration_at IS NULL THEN 1 ELSE 0 END), 0) as pending
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			WHERE %s
		`, whereClause), args...)

		// 7. registration trend (past 30 days or filtered date range)
		var trend []trendpoint
		_ = db.Select(&trend, fmt.Sprintf(`
			SELECT DATE(ep.registration_date) as label, COUNT(*) as value
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			WHERE %s
			GROUP BY label
			ORDER BY label ASC
		`, whereClause), args...)

		// 8. list of recent participants
		type participantitem struct {
			UUID                 string  `db:"uuid" json:"id"`
			ArcherName           string  `db:"archer_name" json:"archer_name"`
			AvatarURL            *string `db:"avatar_url" json:"avatar_url"`
			EventName            string  `db:"event_name" json:"event_name"`
			BowType              *string `db:"bow_type" json:"bow_type"`
			AgeGroup             *string `db:"age_group" json:"age_group"`
			Gender               *string `db:"gender" json:"gender"`
			RegistrationDate     string  `db:"registration_date" json:"registration_date"`
			PaymentStatus        string  `db:"payment_status" json:"payment_status"`
			LastReregistrationAt *string `db:"last_reregistration_at" json:"last_reregistration_at"`
		}
		var recent []participantitem
		_ = db.Select(&recent, fmt.Sprintf(`
			SELECT ep.uuid, COALESCE(a.name, 'tidak ada nama') as archer_name, a.avatar_url, e.name as event_name, 
				   rbt.name as bow_type, rag.name as age_group, rgd.name as gender,
				   ep.registration_date, ep.payment_status, ep.last_reregistration_at
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			WHERE %s
			ORDER BY ep.registration_date DESC
			LIMIT 20
		`, whereClause), args...)

		// mask media URLs
		for i := range recent {
			if recent[i].AvatarURL != nil {
				masked := utils.MaskMediaURL(*recent[i].AvatarURL)
				recent[i].AvatarURL = &masked
			}
		}

		// 9. fetch available events for dropdown filter
		var eventsList []eventoption
		_ = db.Select(&eventsList, `
			SELECT uuid, name, start_date, end_date 
			FROM events 
			WHERE organizer_id = ? 
			ORDER BY start_date DESC
		`, orgID)

		c.JSON(http.StatusOK, gin.H{
			"total_participants":        totalParticipants,
			"gender_split":              genderSplit,
			"bow_type_split":            bowTypeSplit,
			"age_group_split":           ageGroupSplit,
			"registration_source_split": sourceSplit,
			"checked_in_count":          reregStats.CheckedIn,
			"pending_checkin_count":     reregStats.Pending,
			"registration_trend":        trend,
			"recent_participants":       recent,
			"events_list":               eventsList,
		})
	}
}

// getfinancereport returns statistics and details about events finances
func GetOrganizationFinanceReport(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "tidak diizinkan"})
			return
		}

		orgID := userID.(string)

		// filter params
		whereClause := "e.organizer_id = ?"
		args := []interface{}{orgID}

		if eventID := c.Query("event_id"); eventID != "" && eventID != "all" {
			whereClause += " AND e.uuid = ?"
			args = append(args, eventID)
		}

		if startDate := c.Query("start_date"); startDate != "" {
			whereClause += " AND pt.created_at >= ?"
			args = append(args, startDate+" 00:00:00")
		}

		if endDate := c.Query("end_date"); endDate != "" {
			whereClause += " AND pt.created_at <= ?"
			args = append(args, endDate+" 23:59:59")
		}

		if method := c.Query("payment_method"); method != "" && method != "all" {
			whereClause += " AND pt.payment_method = ?"
			args = append(args, method)
		}

		if status := c.Query("status"); status != "" && status != "all" {
			whereClause += " AND pt.status = ?"
			args = append(args, status)
		}

		// 1. revenue summary counts and amounts
		var revSummary struct {
			TotalPaid    float64 `db:"total_paid"`
			TotalPending float64 `db:"total_pending"`
			TotalFailed  float64 `db:"total_failed"`
		}
		err := db.Get(&revSummary, fmt.Sprintf(`
			SELECT 
				COALESCE(SUM(CASE WHEN pt.status = 'paid' THEN pt.amount ELSE 0 END), 0) as total_paid,
				COALESCE(SUM(CASE WHEN pt.status = 'pending' THEN pt.amount ELSE 0 END), 0) as total_pending,
				COALESCE(SUM(CASE WHEN pt.status IN ('expired', 'failed') THEN pt.amount ELSE 0 END), 0) as total_failed
			FROM payment_transactions pt
			JOIN events e ON pt.event_id = e.uuid
			WHERE %s
		`, whereClause), args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal mengambil data keuangan", "details": err.Error()})
			return
		}

		// 2. payment method split (for paid payments)
		var methodSplit []splitamount
		_ = db.Select(&methodSplit, fmt.Sprintf(`
			SELECT COALESCE(pt.payment_method, 'manual') as label, COUNT(*) as count, COALESCE(SUM(pt.amount), 0) as amount
			FROM payment_transactions pt
			JOIN events e ON pt.event_id = e.uuid
			WHERE %s AND pt.status = 'paid'
			GROUP BY label
		`, whereClause), args...)

		// 3. revenue trend
		var trend []trendpoint
		_ = db.Select(&trend, fmt.Sprintf(`
			SELECT DATE(pt.paid_at) as label, COALESCE(SUM(pt.amount), 0) as value
			FROM payment_transactions pt
			JOIN events e ON pt.event_id = e.uuid
			WHERE %s AND pt.status = 'paid' AND pt.paid_at IS NOT NULL
			GROUP BY label
			ORDER BY label ASC
		`, whereClause), args...)

		// 4. recent transaction logs
		type transactionitem struct {
			UUID          string  `db:"uuid" json:"id"`
			Reference     string  `db:"reference" json:"reference"`
			Amount        float64 `db:"amount" json:"amount"`
			PaymentMethod *string `db:"payment_method" json:"payment_method"`
			Status        string  `db:"status" json:"status"`
			CreatedAt     string  `db:"created_at" json:"created_at"`
			PaidAt        *string `db:"paid_at" json:"paid_at"`
			EventName     string  `db:"event_name" json:"event_name"`
			UserName      *string `db:"user_name" json:"user_name"`
			SenderName    *string `db:"sender_name" json:"sender_name"`
		}
		var recent []transactionitem
		_ = db.Select(&recent, fmt.Sprintf(`
			SELECT pt.uuid, pt.reference, pt.amount, pt.payment_method, pt.status, pt.created_at, pt.paid_at,
				   pt.sender_name, e.name as event_name, u.full_name as user_name
			FROM payment_transactions pt
			JOIN events e ON pt.event_id = e.uuid
			LEFT JOIN archers u ON pt.user_id = u.uuid
			WHERE %s
			ORDER BY pt.created_at DESC
			LIMIT 20
		`, whereClause), args...)

		// 5. fetch available events for dropdown filter
		var eventsList []eventoption
		_ = db.Select(&eventsList, `
			SELECT uuid, name, start_date, end_date 
			FROM events 
			WHERE organizer_id = ? 
			ORDER BY start_date DESC
		`, orgID)

		c.JSON(http.StatusOK, gin.H{
			"total_paid":           revSummary.TotalPaid,
			"total_pending":        revSummary.TotalPending,
			"total_failed":         revSummary.TotalFailed,
			"payment_method_split": methodSplit,
			"revenue_trend":        trend,
			"recent_transactions":  recent,
			"events_list":          eventsList,
		})
	}
}

// getperformancereport returns details about event registration capacities and quotas
func GetOrganizationPerformanceReport(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "tidak diizinkan"})
			return
		}

		orgID := userID.(string)

		// filter params
		whereClause := "e.organizer_id = ?"
		args := []interface{}{orgID}

		if startDate := c.Query("start_date"); startDate != "" {
			whereClause += " AND e.start_date >= ?"
			args = append(args, startDate+" 00:00:00")
		}

		if endDate := c.Query("end_date"); endDate != "" {
			whereClause += " AND e.start_date <= ?"
			args = append(args, endDate+" 23:59:59")
		}

		// 1. event status count
		var statusCount []splitcount
		err := db.Select(&statusCount, fmt.Sprintf(`
			SELECT status as label, COUNT(*) as count
			FROM events e
			WHERE %s
			GROUP BY status
		`, whereClause), args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal mengambil data performa event", "details": err.Error()})
			return
		}

		// 2. event details with registrations capacity and fill rates
		type eventperf struct {
			UUID              string  `db:"uuid" json:"id"`
			Name              string  `db:"name" json:"name"`
			Status            string  `db:"status" json:"status"`
			StartDate         *string `db:"start_date" json:"start_date"`
			EndDate           *string `db:"end_date" json:"end_date"`
			TotalCategories   int     `db:"total_categories" json:"total_categories"`
			TotalCapacity     int     `db:"total_capacity" json:"total_capacity"`
			TotalParticipants int     `db:"total_participants" json:"total_participants"`
			FillRate          float64 `json:"fill_rate"`
		}
		var list []eventperf
		_ = db.Select(&list, fmt.Sprintf(`
			SELECT 
				e.uuid, e.name, e.status, e.start_date, e.end_date,
				(SELECT COUNT(*) FROM event_categories ec WHERE ec.event_id = e.uuid AND ec.status = 'active') as total_categories,
				COALESCE((SELECT SUM(ec.max_participants) FROM event_categories ec WHERE ec.event_id = e.uuid AND ec.status = 'active'), 0) as total_capacity,
				(SELECT COUNT(*) FROM event_participants ep WHERE ep.event_id = e.uuid) as total_participants
			FROM events e
			WHERE %s
			ORDER BY e.created_at DESC
		`, whereClause), args...)

		var aggregateCapacity int
		var aggregateParticipants int
		for i := range list {
			aggregateCapacity += list[i].TotalCapacity
			aggregateParticipants += list[i].TotalParticipants
			if list[i].TotalCapacity > 0 {
				list[i].FillRate = (float64(list[i].TotalParticipants) / float64(list[i].TotalCapacity)) * 100
			} else {
				list[i].FillRate = 0
			}
		}

		var averageFillRate float64
		if aggregateCapacity > 0 {
			averageFillRate = (float64(aggregateParticipants) / float64(aggregateCapacity)) * 100
		}

		c.JSON(http.StatusOK, gin.H{
			"status_summary":     statusCount,
			"events_performance": list,
			"average_fill_rate":  averageFillRate,
			"total_capacity":     aggregateCapacity,
			"total_participants": aggregateParticipants,
		})
	}
}

// getattendancereport returns statistics on checked-in vs registered participants
func GetOrganizationAttendanceReport(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "tidak diizinkan"})
			return
		}

		orgID := userID.(string)

		// filter params
		whereClause := "e.organizer_id = ?"
		args := []interface{}{orgID}

		if eventID := c.Query("event_id"); eventID != "" && eventID != "all" {
			whereClause += " AND e.uuid = ?"
			args = append(args, eventID)
		}

		if startDate := c.Query("start_date"); startDate != "" {
			whereClause += " AND ep.registration_date >= ?"
			args = append(args, startDate+" 00:00:00")
		}

		if endDate := c.Query("end_date"); endDate != "" {
			whereClause += " AND ep.registration_date <= ?"
			args = append(args, endDate+" 23:59:59")
		}

		// 1. attendance overview
		var attendanceStats struct {
			TotalRegistered int `db:"total_registered"`
			CheckedIn       int `db:"checked_in"`
			Pending         int `db:"pending"`
		}
		err := db.Get(&attendanceStats, fmt.Sprintf(`
			SELECT 
				COUNT(*) as total_registered,
				COALESCE(SUM(CASE WHEN ep.last_reregistration_at IS NOT NULL THEN 1 ELSE 0 END), 0) as checked_in,
				COALESCE(SUM(CASE WHEN ep.last_reregistration_at IS NULL THEN 1 ELSE 0 END), 0) as pending
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			WHERE %s
		`, whereClause), args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal mengambil data kehadiran", "details": err.Error()})
			return
		}

		// 2. attendance breakdown by category/bow-type/age-group
		type categoryattendance struct {
			BowType    string `db:"bow_type" json:"bow_type"`
			AgeGroup   string `db:"age_group" json:"age_group"`
			Gender     string `db:"gender" json:"gender"`
			Registered int    `db:"registered" json:"registered"`
			CheckedIn  int    `db:"checked_in" json:"checked_in"`
		}
		var categoryList []categoryattendance
		_ = db.Select(&categoryList, fmt.Sprintf(`
			SELECT 
				COALESCE(rbt.name, 'barebow') as bow_type, 
				COALESCE(rag.name, 'umum') as age_group, 
				COALESCE(rgd.name, 'campuran') as gender,
				COUNT(*) as registered,
				COALESCE(SUM(CASE WHEN ep.last_reregistration_at IS NOT NULL THEN 1 ELSE 0 END), 0) as checked_in
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			WHERE %s
			GROUP BY bow_type, age_group, gender
		`, whereClause), args...)

		// 3. check-in timeline trend
		var checkinTrend []trendpoint
		_ = db.Select(&checkinTrend, fmt.Sprintf(`
			SELECT DATE(ep.last_reregistration_at) as label, COUNT(*) as value
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			WHERE %s AND ep.last_reregistration_at IS NOT NULL
			GROUP BY label
			ORDER BY label ASC
		`, whereClause), args...)

		// 4. recent check-ins list
		type checkedinitem struct {
			UUID                 string  `db:"uuid" json:"id"`
			ArcherName           string  `db:"archer_name" json:"archer_name"`
			AvatarURL            *string `db:"avatar_url" json:"avatar_url"`
			EventName            string  `db:"event_name" json:"event_name"`
			BowType              *string `db:"bow_type" json:"bow_type"`
			AgeGroup             *string `db:"age_group" json:"age_group"`
			Gender               *string `db:"gender" json:"gender"`
			LastReregistrationAt string  `db:"last_reregistration_at" json:"last_reregistration_at"`
		}
		var recentCheckedIn []checkedinitem
		_ = db.Select(&recentCheckedIn, fmt.Sprintf(`
			SELECT ep.uuid, COALESCE(a.name, 'tidak ada nama') as archer_name, a.avatar_url, e.name as event_name, 
				   rbt.name as bow_type, rag.name as age_group, rgd.name as gender,
				   ep.last_reregistration_at
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			WHERE %s AND ep.last_reregistration_at IS NOT NULL
			ORDER BY ep.last_reregistration_at DESC
			LIMIT 20
		`, whereClause), args...)

		// mask media URLs
		for i := range recentCheckedIn {
			if recentCheckedIn[i].AvatarURL != nil {
				masked := utils.MaskMediaURL(*recentCheckedIn[i].AvatarURL)
				recentCheckedIn[i].AvatarURL = &masked
			}
		}

		// 5. fetch available events for dropdown filter
		var eventsList []eventoption
		_ = db.Select(&eventsList, `
			SELECT uuid, name, start_date, end_date 
			FROM events 
			WHERE organizer_id = ? 
			ORDER BY start_date DESC
		`, orgID)

		c.JSON(http.StatusOK, gin.H{
			"total_registered":    attendanceStats.TotalRegistered,
			"total_checked_in":    attendanceStats.CheckedIn,
			"total_pending":       attendanceStats.Pending,
			"categories_breakdown": categoryList,
			"checkin_trend":       checkinTrend,
			"recent_checkins":     recentCheckedIn,
			"events_list":         eventsList,
		})
	}
}
