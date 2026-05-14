package mobile

import (
	"Archeris-api/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// MobileGetScorekeeperMe returns current scorekeeper profile
// @Summary Get Scorekeeper Profile
// @Description Get profile information for the authenticated scorekeeper
// @Tags Mobile - Scorekeeper
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MobileScorekeeperMeResponse
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /mobile/scorekeeper/me [get]
func MobileGetScorekeeperMe(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		if userType != "scorekeeper" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Hanya scorekeeper yang bisa mengakses ini"})
			return
		}

		var sk struct {
			UUID             string  `db:"uuid" json:"uuid"`
			OrganizationUUID string  `db:"organization_uuid" json:"organization_uuid"`
			Code             string  `db:"code" json:"code"`
			Name             string  `db:"name" json:"name"`
			Email            *string `db:"email" json:"email"`
			AvatarURL        *string `db:"avatar_url" json:"avatar_url"`
			Status           string  `db:"status" json:"status"`
			OrgName          string  `db:"org_name" json:"organization_name"`
		}

		err := db.Get(&sk, `
			SELECT sk.uuid, sk.organization_uuid, sk.code, sk.name, sk.email, sk.avatar_url, sk.status, o.name as org_name 
			FROM scorekeepers sk 
			JOIN organizations o ON sk.organization_uuid = o.uuid 
			WHERE sk.uuid = ?`, userID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scorekeeper tidak ditemukan"})
			return
		}

		if sk.AvatarURL != nil {
			masked := utils.MaskMediaURL(*sk.AvatarURL)
			sk.AvatarURL = &masked
		}

		c.JSON(http.StatusOK, MobileScorekeeperMeResponse{
			UUID:             sk.UUID,
			OrganizationUUID: sk.OrganizationUUID,
			Code:             sk.Code,
			Name:             sk.Name,
			Email:            sk.Email,
			AvatarURL:        sk.AvatarURL,
			Status:           sk.Status,
			OrganizationName: sk.OrgName,
		})
	}
}

// MobileGetScorekeeperEvents returns events for scorekeeper's organization
// @Summary Get Scorekeeper Events
// @Description Get list of events owned by the scorekeeper's organization
// @Tags Mobile - Scorekeeper
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MobileScorekeeperEventsResponse
// @Router /mobile/scorekeeper/events [get]
func MobileGetScorekeeperEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, _ := c.Get("org_id")
		userType, _ := c.Get("user_type")

		if userType != "scorekeeper" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Hanya scorekeeper yang bisa mengakses ini"})
			return
		}

		var events []MobileEvent
		err := db.Select(&events, `
			SELECT 
				t.uuid, t.name, t.location, t.start_date, t.end_date, t.logo_url, t.banner_url,
				o.name as organizer_name,
				o.avatar_url as organizer_avatar_url,
				(SELECT COUNT(DISTINCT archer_id) FROM event_participants WHERE event_id = t.uuid) as participant_count
			FROM events t
			JOIN organizations o ON t.organizer_id = o.uuid
			WHERE t.organizer_id = ?
			ORDER BY t.start_date DESC`, orgID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data event", "details": err.Error()})
			return
		}

		for i := range events {
			if events[i].LogoURL != nil {
				masked := utils.MaskMediaURL(*events[i].LogoURL)
				events[i].LogoURL = &masked
			}
			if events[i].BannerURL != nil {
				masked := utils.MaskMediaURL(*events[i].BannerURL)
				events[i].BannerURL = &masked
			}
			if events[i].OrganizerAvatarURL != nil {
				masked := utils.MaskMediaURL(*events[i].OrganizerAvatarURL)
				events[i].OrganizerAvatarURL = &masked
			}
		}

		c.JSON(http.StatusOK, MobileScorekeeperEventsResponse{
			Events:     events,
			TotalCount: len(events),
		})
	}
}

