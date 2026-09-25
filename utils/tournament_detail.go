package utils

import (
	"Archeris-api/models"
	"encoding/json"
	"strings"

	"github.com/jmoiron/sqlx"
)

// PopulateEventDetailExtras enriches event detail payload with related data sections.
func PopulateEventDetailExtras(db *sqlx.DB, event *models.EventWithDetails) {
	if event == nil {
		return
	}

	event.LocationDetail = models.EventLocationDetail{
		Venue:        event.Venue,
		Address:      event.Address,
		GmapLink:     event.GmapLink,
		Location:     event.Location,
		City:         event.City,
		LocationType: event.LocationType,
	}

	event.Currency = "IDR"
	if event.Event.Currency != "" {
		event.Currency = event.Event.Currency
	} else if event.Event.PageSettings != nil && *event.Event.PageSettings != "" {
		var ps struct {
			Currency string `json:"currency"`
		}
		if errJson := json.Unmarshal([]byte(*event.Event.PageSettings), &ps); errJson == nil && ps.Currency != "" {
			event.Currency = ps.Currency
		}
	} else if psMap, ok := event.PageSettings.(map[string]interface{}); ok {
		if curr, ok := psMap["currency"].(string); ok && curr != "" {
			event.Currency = curr
		}
	}
	// Fallback to organizer page_settings if tournament currency is still default or empty
	if event.Currency == "IDR" && event.OrganizerID != nil && *event.OrganizerID != "" {
		var pageSettingsStr *string
		err := db.Get(&pageSettingsStr, "SELECT page_settings FROM organizers WHERE uuid = ?", *event.OrganizerID)
		if err == nil && pageSettingsStr != nil && *pageSettingsStr != "" {
			var pageSettings struct {
				Currency string `json:"currency"`
			}
			if errJson := json.Unmarshal([]byte(*pageSettingsStr), &pageSettings); errJson == nil && pageSettings.Currency != "" {
				event.Currency = pageSettings.Currency
			}
		}
	}

	participants := []models.EventParticipantPreview{}
	_ = db.Select(&participants, `
		SELECT
			tp.uuid as participant_id,
			tp.archer_id,
			COALESCE(a.full_name, '') as full_name,
			NULLIF(COALESCE(cl.name, ''), '') as club_name,
			NULLIF(COALESCE(ec.category_name_custom, ag.name, ''), '') as category_name,
			COALESCE(tp.payment_status, 'pending') as payment_status,
			tp.qual_rank,
			tp.qual_score,
			a.avatar_url
		FROM tournament_participants tp
		LEFT JOIN archers a ON tp.archer_id = a.uuid
		LEFT JOIN clubs cl ON a.club_id = cl.uuid
		LEFT JOIN tournament_categories ec ON tp.category_id = ec.uuid
		LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
		WHERE tp.tournament_id = ?
		ORDER BY tp.registration_date DESC
		LIMIT 20
	`, event.UUID)
	for i := range participants {
		if participants[i].AvatarURL != nil {
			masked := MaskMediaURL(*participants[i].AvatarURL)
			participants[i].AvatarURL = &masked
		}
	}
	event.Participants = participants

	schedules := []models.EventSchedule{}
	_ = db.Select(&schedules, `
		SELECT 
			uuid,
			tournament_id as event_id,
			title,
			subtitle,
			description,
			item_type,
			CAST(start_time AS CHAR) AS start_time,
			CAST(end_time AS CHAR) AS end_time,
			duration_minutes,
			day_number,
			day_number as day_order,
			CAST(schedule_date AS CHAR) as schedule_date,
			location,
			session_code,
			target_start,
			target_end,
			elim_round,
			sort_order,
			created_at,
			updated_at
		FROM tournament_schedule_items
		WHERE tournament_id = ?
		ORDER BY COALESCE(NULLIF(schedule_date, ''), '9999-12-31') ASC, day_number ASC, start_time ASC, sort_order ASC
	`, event.UUID)

	if len(schedules) == 0 {
		_ = db.Select(&schedules, `
			SELECT 
				uuid,
				tournament_id as event_id,
				title,
				description,
				'general' as item_type,
				CAST(start_time AS CHAR) as start_time,
				CAST(end_time AS CHAR) as end_time,
				60 as duration_minutes,
				COALESCE(day_order, 1) as day_order,
				COALESCE(day_order, 1) as day_number,
				location,
				sort_order,
				created_at,
				updated_at
			FROM tournament_schedules
			WHERE tournament_id = ?
			ORDER BY COALESCE(day_order, 0), COALESCE(sort_order, 0), start_time
		`, event.UUID)
	}
	event.Schedules = schedules

	results := []models.EventResultPreview{}
	_ = db.Select(&results, `
		SELECT
			tp.uuid as participant_id,
			COALESCE(a.full_name, '') as full_name,
			NULLIF(COALESCE(ec.category_name_custom, ag.name, ''), '') as category_name,
			tp.qual_rank as rank,
			tp.qual_score as score,
			COALESCE(scores.total_x, 0) as x_count
		FROM tournament_participants tp
		LEFT JOIN archers a ON tp.archer_id = a.uuid
		LEFT JOIN tournament_categories ec ON tp.category_id = ec.uuid
		LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
		LEFT JOIN (
			SELECT participant_uuid, SUM(x_count_end) as total_x
			FROM qualification_end_scores
			GROUP BY participant_uuid
		) scores ON tp.uuid = scores.participant_uuid
		WHERE tp.tournament_id = ? AND tp.qual_rank IS NOT NULL
		ORDER BY tp.qual_rank ASC, tp.qual_score DESC
		LIMIT 20
	`, event.UUID)
	event.Results = results

	gallery := []models.EventImage{}
	_ = db.Select(&gallery, `
		SELECT uuid, tournament_id, url, caption, alt_text, display_order, is_primary, created_at
		FROM tournament_images
		WHERE tournament_id = ?
		ORDER BY display_order, created_at
	`, event.UUID)
	for i := range gallery {
		gallery[i].URL = MaskMediaURL(gallery[i].URL)
	}
	event.Gallery = gallery

	competitionCategories := []models.EventCompetitionCategory{}
	_ = db.Select(&competitionCategories, `
		SELECT
			ec.uuid as category_id,
			NULLIF(COALESCE(bt.name, ''), '') as division_name,
			NULLIF(COALESCE(ec.category_name_custom, ag.name, ''), '') as category_name,
			NULLIF(COALESCE(et.name, ''), '') as event_type_name,
			NULLIF(COALESCE(gd.name, ''), '') as gender_division_name,
			COUNT(tp.uuid) as participant_count,
			COALESCE(
				NULLIF(ec.fee, 0.00),
				CASE 
					WHEN t.fee_mode = 'per_type' AND LOWER(COALESCE(et.name, '')) LIKE '%mixed%' THEN NULLIF(t.fee_mixed_team, 0.00)
					WHEN t.fee_mode = 'per_type' AND (LOWER(COALESCE(et.name, '')) LIKE '%team%' OR LOWER(COALESCE(et.name, '')) LIKE '%beregu%') THEN NULLIF(t.fee_team, 0.00)
					WHEN t.fee_mode = 'per_type' THEN NULLIF(t.fee_individual, 0.00)
					ELSE NULLIF(t.entry_fee, 0.00)
				END,
				t.entry_fee,
				0.00
			) as fee
		FROM tournament_categories ec
		JOIN tournaments t ON ec.tournament_id = t.uuid
		LEFT JOIN tournament_participants tp ON tp.category_id = ec.uuid
		LEFT JOIN ref_bow_types bt ON ec.division_uuid = bt.uuid
		LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
		LEFT JOIN ref_tournament_types et ON ec.tournament_type_uuid = et.uuid
		LEFT JOIN ref_gender_divisions gd ON ec.gender_division_uuid = gd.uuid
		WHERE ec.tournament_id = ?
		GROUP BY ec.uuid, bt.name, ec.category_name_custom, ag.name, et.name, gd.name, ec.fee, t.fee_mode, t.fee_individual, t.fee_team, t.fee_mixed_team, t.entry_fee
		ORDER BY participant_count DESC, ec.created_at ASC
	`, event.UUID)

	// Fallback check: if category fee is 0 but page_settings has fee_per_category or fee_per_type
	if event.Event.PageSettings != nil && *event.Event.PageSettings != "" {
		var ps struct {
			FeeMode        string             `json:"fee_mode"`
			FeePerType     map[string]float64 `json:"fee_per_type"`
			FeePerCategory map[string]float64 `json:"fee_per_category"`
		}
		if errJson := json.Unmarshal([]byte(*event.Event.PageSettings), &ps); errJson == nil {
			for i := range competitionCategories {
				catID := competitionCategories[i].CategoryID
				if (competitionCategories[i].Fee == nil || *competitionCategories[i].Fee == 0) {
					if ps.FeeMode == "per_category" && ps.FeePerCategory != nil {
						if feeVal, ok := ps.FeePerCategory[catID]; ok && feeVal > 0 {
							feeCopy := feeVal
							competitionCategories[i].Fee = &feeCopy
						}
					} else if ps.FeeMode == "per_type" && ps.FeePerType != nil {
						eventTypeName := ""
						if competitionCategories[i].EventTypeName != nil {
							eventTypeName = *competitionCategories[i].EventTypeName
						}
						typeKey := "individual"
						if strings.Contains(strings.ToLower(eventTypeName), "mixed") {
							typeKey = "mixed_team"
						} else if strings.Contains(strings.ToLower(eventTypeName), "team") || strings.Contains(strings.ToLower(eventTypeName), "beregu") {
							typeKey = "team"
						}
						if feeVal, ok := ps.FeePerType[typeKey]; ok && feeVal > 0 {
							feeCopy := feeVal
							competitionCategories[i].Fee = &feeCopy
						}
					}
				}
			}
		}
	}

	event.CompetitionCategories = competitionCategories

	// Parse JSON fields
	if event.Event.PageSettings != nil && *event.Event.PageSettings != "" {
		_ = json.Unmarshal([]byte(*event.Event.PageSettings), &event.PageSettings)
	}
	if event.Event.FAQ != nil && *event.Event.FAQ != "" {
		_ = json.Unmarshal([]byte(*event.Event.FAQ), &event.FAQ)
	}
}

