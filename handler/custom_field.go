package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"Archeris-api/models"
	"Archeris-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// resolveTournamentUUID finds the tournament UUID and checks ownership if required
func resolveTournamentUUID(db *sqlx.DB, idOrSlug string) (string, string, error) {
	var tour struct {
		UUID        string `db:"uuid"`
		OrganizerID string `db:"organizer_id"`
	}
	err := db.Get(&tour, "SELECT uuid, COALESCE(organizer_id, '') as organizer_id FROM tournaments WHERE uuid = ? OR slug = ? OR code = ? LIMIT 1", idOrSlug, idOrSlug, idOrSlug)
	if err != nil {
		return "", "", err
	}
	return tour.UUID, tour.OrganizerID, nil
}

// checkOrganizerAuth verifies if the current authenticated user can modify this tournament
func checkOrganizerAuth(c *gin.Context, organizerID string) bool {
	userRole, _ := c.Get("role")
	if userRole == "admin" {
		return true
	}
	orgID, exists := c.Get("org_id")
	if exists && orgID != nil && fmt.Sprintf("%v", orgID) == organizerID {
		return true
	}
	userID, existsUser := c.Get("user_id")
	if existsUser && userID != nil && fmt.Sprintf("%v", userID) == organizerID {
		return true
	}
	return false
}

// GetTournamentCustomFields retrieves all custom registration fields & layout blocks for a tournament (Organizer / Admin)
func GetTournamentCustomFields(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentParam := c.Param("id")
		if tournamentParam == "" {
			tournamentParam = c.Param("slug")
		}
		tourUUID, _, err := resolveTournamentUUID(db, tournamentParam)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan"})
			return
		}

		var fields []models.TournamentCustomField
		query := `
			SELECT id, uuid, tournament_uuid, field_key, label_id, label_en, 
			       placeholder_id, placeholder_en, description_id, description_en,
			       field_type, options_json, is_required, is_active, 
			       applies_to_category_ids, display_order, col_span, element_type, 
			       style_config, created_at, updated_at
			FROM tournament_custom_fields
			WHERE tournament_uuid = ?
			ORDER BY display_order ASC, id ASC
		`
		err = db.Select(&fields, query, tourUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data formulir", "details": err.Error()})
			return
		}

		for i := range fields {
			fields[i].ParseJSONFields()
		}

		c.JSON(http.StatusOK, gin.H{
			"tournament_uuid": tourUUID,
			"total":           len(fields),
			"fields":          fields,
		})
	}
}

// GetPublicTournamentCustomFields retrieves ACTIVE custom fields and layout blocks for registration form
func GetPublicTournamentCustomFields(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentParam := c.Param("id")
		if tournamentParam == "" {
			tournamentParam = c.Param("slug")
		}
		tourUUID, _, err := resolveTournamentUUID(db, tournamentParam)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan"})
			return
		}

		var fields []models.TournamentCustomField
		query := `
			SELECT id, uuid, tournament_uuid, field_key, label_id, label_en, 
			       placeholder_id, placeholder_en, description_id, description_en,
			       field_type, options_json, is_required, is_active, 
			       applies_to_category_ids, display_order, col_span, element_type, 
			       style_config, created_at, updated_at
			FROM tournament_custom_fields
			WHERE tournament_uuid = ? AND is_active = TRUE
			ORDER BY display_order ASC, id ASC
		`
		err = db.Select(&fields, query, tourUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil field pendaftaran", "details": err.Error()})
			return
		}

		for i := range fields {
			fields[i].ParseJSONFields()
		}

		c.JSON(http.StatusOK, gin.H{
			"tournament_uuid": tourUUID,
			"total":           len(fields),
			"fields":          fields,
		})
	}
}

// CreateTournamentCustomField creates a new field or layout block (heading, divider, spacer, notice)
func CreateTournamentCustomField(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentParam := c.Param("id")
		tourUUID, organizerID, err := resolveTournamentUUID(db, tournamentParam)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan"})
			return
		}

		if !checkOrganizerAuth(c, organizerID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki izin untuk mengelola formulir turnamen ini"})
			return
		}

		var input models.CustomFieldInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payload tidak valid", "details": err.Error()})
			return
		}

		elementType := "field"
		if input.ElementType != nil && *input.ElementType != "" {
			elementType = *input.ElementType
		}

		fieldType := input.FieldType
		if fieldType == "" {
			fieldType = "text"
		}

		fieldKey := strings.ToLower(strings.TrimSpace(input.FieldKey))
		if fieldKey == "" {
			if elementType != "field" {
				fieldKey = fmt.Sprintf("%s_%s", elementType, uuid.New().String()[:8])
			} else {
				fieldKey = utils.CleanSlug(input.LabelID)
				if fieldKey == "" {
					fieldKey = "field_" + uuid.New().String()[:8]
				}
			}
		}

		// Ensure field_key is unique within this tournament
		var keyCount int
		_ = db.Get(&keyCount, "SELECT COUNT(*) FROM tournament_custom_fields WHERE tournament_uuid = ? AND field_key = ?", tourUUID, fieldKey)
		if keyCount > 0 {
			fieldKey = fmt.Sprintf("%s_%s", fieldKey, uuid.New().String()[:4])
		}

		colSpan := 12
		if input.ColSpan != nil && *input.ColSpan > 0 {
			colSpan = *input.ColSpan
		}

		// Calculate display_order if not set
		displayOrder := 0
		if input.DisplayOrder != nil {
			displayOrder = *input.DisplayOrder
		} else {
			var maxOrder *int
			_ = db.Get(&maxOrder, "SELECT MAX(display_order) FROM tournament_custom_fields WHERE tournament_uuid = ?", tourUUID)
			if maxOrder != nil {
				displayOrder = *maxOrder + 1
			} else {
				displayOrder = 1
			}
		}

		// Marshal options JSON
		var optionsJSON *string
		if len(input.Options) > 0 {
			bytes, err := json.Marshal(input.Options)
			if err == nil {
				str := string(bytes)
				optionsJSON = &str
			}
		}

		// Marshal applies_to_category_ids JSON
		var categoryIDsJSON *string
		if len(input.AppliesToCategoryIDs) > 0 {
			bytes, err := json.Marshal(input.AppliesToCategoryIDs)
			if err == nil {
				str := string(bytes)
				categoryIDsJSON = &str
			}
		}

		// Marshal style_config JSON
		var styleConfigJSON *string
		if len(input.StyleConfig) > 0 {
			bytes, err := json.Marshal(input.StyleConfig)
			if err == nil {
				str := string(bytes)
				styleConfigJSON = &str
			}
		}

		isActive := true
		if input.IsActive != nil {
			isActive = *input.IsActive
		}

		newUUID := uuid.New().String()
		query := `
			INSERT INTO tournament_custom_fields (
				uuid, tournament_uuid, field_key, label_id, label_en,
				placeholder_id, placeholder_en, description_id, description_en,
				field_type, options_json, is_required, is_active,
				applies_to_category_ids, display_order, col_span, element_type, style_config,
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
		`
		_, err = db.Exec(query,
			newUUID, tourUUID, fieldKey, input.LabelID, input.LabelEN,
			input.PlaceholderID, input.PlaceholderEN, input.DescriptionID, input.DescriptionEN,
			fieldType, optionsJSON, input.IsRequired, isActive,
			categoryIDsJSON, displayOrder, colSpan, elementType, styleConfigJSON,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan elemen formulir", "details": err.Error()})
			return
		}

		// Return created field
		var created models.TournamentCustomField
		_ = db.Get(&created, `
			SELECT id, uuid, tournament_uuid, field_key, label_id, label_en, 
			       placeholder_id, placeholder_en, description_id, description_en,
			       field_type, options_json, is_required, is_active, 
			       applies_to_category_ids, display_order, col_span, element_type, style_config,
			       created_at, updated_at
			FROM tournament_custom_fields WHERE uuid = ?
		`, newUUID)
		created.ParseJSONFields()

		c.JSON(http.StatusCreated, gin.H{
			"message": "Elemen formulir berhasil ditambahkan",
			"field":   created,
		})
	}
}

// UpdateTournamentCustomField updates an existing field or layout block configuration
func UpdateTournamentCustomField(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentParam := c.Param("id")
		fieldParam := c.Param("fieldId")

		tourUUID, organizerID, err := resolveTournamentUUID(db, tournamentParam)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan"})
			return
		}

		if !checkOrganizerAuth(c, organizerID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki izin untuk mengelola formulir turnamen ini"})
			return
		}

		var input models.CustomFieldInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payload tidak valid", "details": err.Error()})
			return
		}

		// Check field existence
		var existingField models.TournamentCustomField
		err = db.Get(&existingField, "SELECT id, uuid, field_key, display_order, is_active, col_span, element_type, field_type FROM tournament_custom_fields WHERE (uuid = ? OR id = ?) AND tournament_uuid = ?", fieldParam, fieldParam, tourUUID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Elemen formulir tidak ditemukan"})
			return
		}

		// Marshal options JSON
		var optionsJSON *string
		if len(input.Options) > 0 {
			bytes, err := json.Marshal(input.Options)
			if err == nil {
				str := string(bytes)
				optionsJSON = &str
			}
		}

		// Marshal applies_to_category_ids JSON
		var categoryIDsJSON *string
		if len(input.AppliesToCategoryIDs) > 0 {
			bytes, err := json.Marshal(input.AppliesToCategoryIDs)
			if err == nil {
				str := string(bytes)
				categoryIDsJSON = &str
			}
		}

		// Marshal style_config JSON
		var styleConfigJSON *string
		if len(input.StyleConfig) > 0 {
			bytes, err := json.Marshal(input.StyleConfig)
			if err == nil {
				str := string(bytes)
				styleConfigJSON = &str
			}
		}

		isActive := existingField.IsActive
		if input.IsActive != nil {
			isActive = *input.IsActive
		}

		displayOrder := existingField.DisplayOrder
		if input.DisplayOrder != nil {
			displayOrder = *input.DisplayOrder
		}

		colSpan := existingField.ColSpan
		if input.ColSpan != nil && *input.ColSpan > 0 {
			colSpan = *input.ColSpan
		}

		elementType := existingField.ElementType
		if input.ElementType != nil && *input.ElementType != "" {
			elementType = *input.ElementType
		}

		fieldType := existingField.FieldType
		if input.FieldType != "" {
			fieldType = input.FieldType
		}

		fieldKey := existingField.FieldKey
		if strings.TrimSpace(input.FieldKey) != "" {
			fieldKey = strings.TrimSpace(input.FieldKey)
		}

		query := `
			UPDATE tournament_custom_fields SET
				field_key = ?,
				label_id = ?,
				label_en = ?,
				placeholder_id = ?,
				placeholder_en = ?,
				description_id = ?,
				description_en = ?,
				field_type = ?,
				options_json = ?,
				is_required = ?,
				is_active = ?,
				applies_to_category_ids = ?,
				display_order = ?,
				col_span = ?,
				element_type = ?,
				style_config = ?,
				updated_at = NOW()
			WHERE uuid = ? AND tournament_uuid = ?
		`
		_, err = db.Exec(query,
			fieldKey, input.LabelID, input.LabelEN,
			input.PlaceholderID, input.PlaceholderEN, input.DescriptionID, input.DescriptionEN,
			fieldType, optionsJSON, input.IsRequired, isActive,
			categoryIDsJSON, displayOrder, colSpan, elementType, styleConfigJSON,
			existingField.UUID, tourUUID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui elemen formulir", "details": err.Error()})
			return
		}

		var updated models.TournamentCustomField
		_ = db.Get(&updated, `
			SELECT id, uuid, tournament_uuid, field_key, label_id, label_en, 
			       placeholder_id, placeholder_en, description_id, description_en,
			       field_type, options_json, is_required, is_active, 
			       applies_to_category_ids, display_order, col_span, element_type, style_config,
			       created_at, updated_at
			FROM tournament_custom_fields WHERE uuid = ?
		`, existingField.UUID)
		updated.ParseJSONFields()

		c.JSON(http.StatusOK, gin.H{
			"message": "Elemen formulir berhasil diperbarui",
			"field":   updated,
		})
	}
}

// DeleteTournamentCustomField deletes a custom field or layout block
func DeleteTournamentCustomField(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentParam := c.Param("id")
		fieldParam := c.Param("fieldId")

		tourUUID, organizerID, err := resolveTournamentUUID(db, tournamentParam)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan"})
			return
		}

		if !checkOrganizerAuth(c, organizerID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki izin untuk menghapus elemen pada turnamen ini"})
			return
		}

		// Delete participant values first, then field
		var fieldUUID string
		err = db.Get(&fieldUUID, "SELECT uuid FROM tournament_custom_fields WHERE (uuid = ? OR id = ?) AND tournament_uuid = ?", fieldParam, fieldParam, tourUUID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Elemen formulir tidak ditemukan"})
			return
		}

		_, _ = db.Exec("DELETE FROM participant_custom_field_values WHERE field_uuid = ?", fieldUUID)
		_, err = db.Exec("DELETE FROM tournament_custom_fields WHERE uuid = ? AND tournament_uuid = ?", fieldUUID, tourUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus elemen formulir", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Elemen formulir berhasil dihapus"})
	}
}

// ReorderTournamentCustomFields updates the display order for multiple fields in a batch
func ReorderTournamentCustomFields(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentParam := c.Param("id")
		tourUUID, organizerID, err := resolveTournamentUUID(db, tournamentParam)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan"})
			return
		}

		if !checkOrganizerAuth(c, organizerID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki izin untuk mengubah urutan elemen"})
			return
		}

		var req models.CustomFieldReorderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format payload reorder tidak valid", "details": err.Error()})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		for _, item := range req.Orders {
			_, err := tx.Exec("UPDATE tournament_custom_fields SET display_order = ? WHERE (uuid = ? OR id = ?) AND tournament_uuid = ?", item.DisplayOrder, item.FieldUUID, item.FieldUUID, tourUUID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui urutan", "details": err.Error()})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan urutan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Urutan elemen formulir berhasil diperbarui"})
	}
}

// Helper: SaveParticipantCustomFields inserts or updates submitted custom field values for given participant (skips non-field layout blocks)
func SaveParticipantCustomFields(tx *sqlx.Tx, tournamentUUID, participantUUID string, customValues map[string]interface{}) error {
	if len(customValues) == 0 {
		return nil
	}

	// Fetch only actual data fields (element_type = 'field') defined for this tournament
	var fields []struct {
		UUID     string `db:"uuid"`
		FieldKey string `db:"field_key"`
	}
	err := tx.Select(&fields, "SELECT uuid, field_key FROM tournament_custom_fields WHERE tournament_uuid = ? AND (element_type = 'field' OR element_type IS NULL OR element_type = '')", tournamentUUID)
	if err != nil || len(fields) == 0 {
		return nil
	}

	keyToUUID := make(map[string]string)
	for _, f := range fields {
		keyToUUID[f.FieldKey] = f.UUID
		keyToUUID[f.UUID] = f.UUID
	}

	for k, v := range customValues {
		fieldUUID, exists := keyToUUID[k]
		if !exists || v == nil {
			continue
		}

		var strValue string
		switch val := v.(type) {
		case string:
			strValue = val
		case []interface{}:
			bytes, _ := json.Marshal(val)
			strValue = string(bytes)
		case map[string]interface{}:
			bytes, _ := json.Marshal(val)
			strValue = string(bytes)
		default:
			strValue = fmt.Sprintf("%v", val)
		}

		if strings.TrimSpace(strValue) == "" {
			continue
		}

		valUUID := uuid.New().String()
		query := `
			INSERT INTO participant_custom_field_values (
				uuid, participant_uuid, field_uuid, field_value, created_at, updated_at
			) VALUES (?, ?, ?, ?, NOW(), NOW())
			ON DUPLICATE KEY UPDATE field_value = VALUES(field_value), updated_at = NOW()
		`
		_, err := tx.Exec(query, valUUID, participantUUID, fieldUUID, strValue)
		if err != nil {
			return err
		}
	}

	return nil
}

// Helper: GetParticipantCustomFieldValues retrieves all answered custom fields for a participant
func GetParticipantCustomFieldValues(db *sqlx.DB, participantUUID string) ([]models.CustomFieldValueDetail, error) {
	query := `
		SELECT 
			tcf.uuid as field_uuid,
			tcf.field_key,
			tcf.label_id,
			tcf.label_en,
			tcf.field_type,
			pcfv.field_value,
			tcf.options_json,
			tcf.is_required,
			tcf.display_order,
			tcf.col_span,
			tcf.element_type
		FROM participant_custom_field_values pcfv
		JOIN tournament_custom_fields tcf ON pcfv.field_uuid = tcf.uuid
		WHERE pcfv.participant_uuid = ?
		ORDER BY tcf.display_order ASC, tcf.id ASC
	`
	type tempRow struct {
		FieldUUID    string  `db:"field_uuid"`
		FieldKey     string  `db:"field_key"`
		LabelID      string  `db:"label_id"`
		LabelEN      *string `db:"label_en"`
		FieldType    string  `db:"field_type"`
		FieldValue   *string `db:"field_value"`
		OptionsJSON  *string `db:"options_json"`
		IsRequired   bool    `db:"is_required"`
		DisplayOrder int     `db:"display_order"`
		ColSpan      int     `db:"col_span"`
		ElementType  string  `db:"element_type"`
	}

	var rows []tempRow
	err := db.Select(&rows, query, participantUUID)
	if err != nil {
		return nil, err
	}

	res := make([]models.CustomFieldValueDetail, 0, len(rows))
	for _, r := range rows {
		item := models.CustomFieldValueDetail{
			FieldUUID:    r.FieldUUID,
			FieldKey:     r.FieldKey,
			LabelID:      r.LabelID,
			LabelEN:      r.LabelEN,
			FieldType:    r.FieldType,
			FieldValue:   r.FieldValue,
			IsRequired:   r.IsRequired,
			DisplayOrder: r.DisplayOrder,
			ColSpan:      r.ColSpan,
			ElementType:  r.ElementType,
		}
		if r.OptionsJSON != nil && *r.OptionsJSON != "" {
			var opts []string
			if err := json.Unmarshal([]byte(*r.OptionsJSON), &opts); err == nil {
				item.Options = opts
			}
		}
		res = append(res, item)
	}

	return res, nil
}
