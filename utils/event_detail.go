package utils

import (
	"archeryhub-api/models"

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

	participants := []models.EventParticipantPreview{}
	_ = db.Select(&participants, `
		SELECT
			tp.uuid as participant_id,
			tp.archer_id,
			COALESCE(a.full_name, '') as full_name,
			NULLIF(COALESCE(cl.name, ''), '') as club_name,
			NULLIF(COALESCE(ec.category_name_custom, ag.name, ''), '') as category_name,
			COALESCE(tp.payment_status, 'menunggu acc') as payment_status,
			tp.qual_rank,
			tp.qual_score,
			a.avatar_url
		FROM event_participants tp
		LEFT JOIN archers a ON tp.archer_id = a.uuid
		LEFT JOIN clubs cl ON a.club_id = cl.uuid
		LEFT JOIN event_categories ec ON tp.category_id = ec.uuid
		LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
		WHERE tp.event_id = ?
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
		SELECT uuid, event_id, title, description, start_time, end_time, day_order, sort_order, location, created_at, updated_at
		FROM event_schedule
		WHERE event_id = ?
		ORDER BY COALESCE(day_order, 0), COALESCE(sort_order, 0), start_time
	`, event.UUID)
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
		FROM event_participants tp
		LEFT JOIN archers a ON tp.archer_id = a.uuid
		LEFT JOIN event_categories ec ON tp.category_id = ec.uuid
		LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
		LEFT JOIN (
			SELECT participant_uuid, SUM(x_count_end) as total_x
			FROM qualification_end_scores
			GROUP BY participant_uuid
		) scores ON tp.uuid = scores.participant_uuid
		WHERE tp.event_id = ? AND tp.qual_rank IS NOT NULL
		ORDER BY tp.qual_rank ASC, tp.qual_score DESC
		LIMIT 20
	`, event.UUID)
	event.Results = results

	gallery := []models.EventImage{}
	_ = db.Select(&gallery, `
		SELECT uuid, event_id, url, caption, alt_text, display_order, is_primary, created_at
		FROM event_images
		WHERE event_id = ?
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
			COUNT(tp.uuid) as participant_count
		FROM event_categories ec
		LEFT JOIN event_participants tp ON tp.category_id = ec.uuid
		LEFT JOIN ref_bow_types bt ON ec.division_uuid = bt.uuid
		LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
		LEFT JOIN ref_event_types et ON ec.event_type_uuid = et.uuid
		LEFT JOIN ref_gender_divisions gd ON ec.gender_division_uuid = gd.uuid
		WHERE ec.event_id = ?
		GROUP BY ec.uuid, bt.name, ec.category_name_custom, ag.name, et.name, gd.name
		ORDER BY participant_count DESC, ec.created_at ASC
	`, event.UUID)
	event.CompetitionCategories = competitionCategories
}
