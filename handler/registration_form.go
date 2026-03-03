package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// ─── Models ──────────────────────────────────────────────────────────────────

type RegistrationForm struct {
	UUID        string     `json:"uuid" db:"uuid"`
	ClubID      string     `json:"club_id" db:"club_id"`
	Title       string     `json:"title" db:"title"`
	Description *string    `json:"description" db:"description"`
	Theme       *string    `json:"theme" db:"theme"`
	IsPublished bool       `json:"is_published" db:"is_published"`
	Settings    *string    `json:"settings" db:"settings"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

type FormSection struct {
	UUID        string     `json:"uuid" db:"uuid"`
	FormID      string     `json:"form_id" db:"form_id"`
	Title       string     `json:"title" db:"title"`
	Description *string    `json:"description" db:"description"`
	OrderIndex  int        `json:"order_index" db:"order_index"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	Fields      []FormField `json:"fields"`
}

type FormField struct {
	UUID        string     `json:"uuid" db:"uuid"`
	SectionID   string     `json:"section_id" db:"section_id"`
	FormID      string     `json:"form_id" db:"form_id"`
	FieldType   string     `json:"field_type" db:"field_type"`
	Label       string     `json:"label" db:"label"`
	Placeholder *string    `json:"placeholder" db:"placeholder"`
	HelperText  *string    `json:"helper_text" db:"helper_text"`
	IsRequired  bool       `json:"is_required" db:"is_required"`
	Options     *string    `json:"options" db:"options"`
	Validation  *string    `json:"validation" db:"validation"`
	MapToField  *string    `json:"map_to_field" db:"map_to_field"`
	OrderIndex  int        `json:"order_index" db:"order_index"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

type FormWithSections struct {
	RegistrationForm
	Sections []FormSection `json:"sections"`
}

// ─── Helper ───────────────────────────────────────────────────────────────────

func getClubID(c *gin.Context) string {
	userID, _ := c.Get("user_id")
	return fmt.Sprintf("%v", userID)
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

// GetRegistrationForm returns the club's form (with all sections and fields)
func GetRegistrationForm(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubID(c)

		var form RegistrationForm
		err := db.Get(&form, `SELECT * FROM club_registration_forms WHERE club_id = ? LIMIT 1`, clubID)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"data": nil})
			return
		}

		result, err := buildFormWithSections(db, form)
		if err != nil {
			logrus.WithError(err).Error("[FORM] Failed to build form with sections")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load form"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

// GetPublicRegistrationForm returns the published form for a given club (for join page)
func GetPublicRegistrationForm(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := c.Param("slug")

		var form RegistrationForm
		err := db.Get(&form, `SELECT * FROM club_registration_forms WHERE club_id = ? AND is_published = 1 LIMIT 1`, clubID)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"data": nil})
			return
		}

		result, err := buildFormWithSections(db, form)
		if err != nil {
			logrus.WithError(err).Error("[FORM] Failed to build public form")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load form"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

// CreateRegistrationForm creates a new form for the club
func CreateRegistrationForm(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubID(c)

		// Only one form per club
		var count int
		_ = db.Get(&count, `SELECT COUNT(*) FROM club_registration_forms WHERE club_id = ?`, clubID)
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Form already exists for this club"})
			return
		}

		var req struct {
			Title       string  `json:"title"`
			Description *string `json:"description"`
		}
		c.ShouldBindJSON(&req)

		if req.Title == "" {
			req.Title = "Form Pendaftaran Anggota"
		}

		formID := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO club_registration_forms (uuid, club_id, title, description, is_published)
			VALUES (?, ?, ?, ?, 1)
		`, formID, clubID, req.Title, req.Description)
		if err != nil {
			logrus.WithError(err).Error("[FORM] Failed to create registration form")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create form"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"id": formID, "message": "Form created successfully"})
	}
}

// UpdateRegistrationForm updates form metadata (title, description, theme, settings)
func UpdateRegistrationForm(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubID(c)
		formID := c.Param("formId")

		var req struct {
			Title       *string `json:"title"`
			Description *string `json:"description"`
			Theme       *string `json:"theme"`
			Settings    *string `json:"settings"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := db.Exec(`
			UPDATE club_registration_forms 
			SET title = COALESCE(?, title),
			    description = COALESCE(?, description),
			    theme = COALESCE(?, theme),
			    settings = COALESCE(?, settings),
			    updated_at = NOW()
			WHERE uuid = ? AND club_id = ?
		`, req.Title, req.Description, req.Theme, req.Settings, formID, clubID)

		if err != nil {
			logrus.WithError(err).Error("[FORM] Failed to update registration form")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update form"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Form updated"})
	}
}

// PublishRegistrationForm toggles the publish status
func PublishRegistrationForm(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubID(c)
		formID := c.Param("formId")

		var req struct {
			IsPublished bool `json:"is_published"`
		}
		c.ShouldBindJSON(&req)

		_, err := db.Exec(`
			UPDATE club_registration_forms SET is_published = ?, updated_at = NOW()
			WHERE uuid = ? AND club_id = ?
		`, req.IsPublished, formID, clubID)

		if err != nil {
			logrus.WithError(err).Error("[FORM] Failed to publish registration form")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update publish status"})
			return
		}

		status := "published"
		if !req.IsPublished {
			status = "unpublished"
		}
		c.JSON(http.StatusOK, gin.H{"message": "Form " + status})
	}
}

// DeleteRegistrationForm deletes the entire form
func DeleteRegistrationForm(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubID(c)
		formID := c.Param("formId")

		_, err := db.Exec(`DELETE FROM club_registration_forms WHERE uuid = ? AND club_id = ?`, formID, clubID)
		if err != nil {
			logrus.WithError(err).Error("[FORM] Failed to delete registration form")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete form"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Form deleted"})
	}
}

// ─── Section Handlers ─────────────────────────────────────────────────────────

// CreateFormSection adds a section to a form
func CreateFormSection(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubID(c)
		formID := c.Param("formId")

		// Verify ownership
		if !verifyFormOwnership(db, formID, clubID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
			return
		}

		var req struct {
			Title       string  `json:"title" binding:"required"`
			Description *string `json:"description"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get next order index
		var maxOrder int
		_ = db.Get(&maxOrder, `SELECT COALESCE(MAX(order_index), -1) FROM club_form_sections WHERE form_id = ?`, formID)

		sectionID := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO club_form_sections (uuid, form_id, title, description, order_index)
			VALUES (?, ?, ?, ?, ?)
		`, sectionID, formID, req.Title, req.Description, maxOrder+1)

		if err != nil {
			logrus.WithError(err).Error("[FORM] Failed to create section")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create section"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"id": sectionID, "message": "Section created"})
	}
}

// UpdateFormSection updates a section's title/description
func UpdateFormSection(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubID(c)
		formID := c.Param("formId")
		sectionID := c.Param("sectionId")

		if !verifyFormOwnership(db, formID, clubID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
			return
		}

		var req struct {
			Title       *string `json:"title"`
			Description *string `json:"description"`
		}
		c.ShouldBindJSON(&req)

		_, err := db.Exec(`
			UPDATE club_form_sections
			SET title = COALESCE(?, title), description = COALESCE(?, description), updated_at = NOW()
			WHERE uuid = ? AND form_id = ?
		`, req.Title, req.Description, sectionID, formID)

		if err != nil {
			logrus.WithError(err).Error("[FORM] Failed to update section")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update section"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Section updated"})
	}
}

// DeleteFormSection deletes a section and all its fields
func DeleteFormSection(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubID(c)
		formID := c.Param("formId")
		sectionID := c.Param("sectionId")

		if !verifyFormOwnership(db, formID, clubID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
			return
		}

		_, err := db.Exec(`DELETE FROM club_form_sections WHERE uuid = ? AND form_id = ?`, sectionID, formID)
		if err != nil {
			logrus.WithError(err).Error("[FORM] Failed to delete section")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete section"})
			return
		}

		// Also delete fields in this section
		db.Exec(`DELETE FROM club_form_fields WHERE section_id = ?`, sectionID)

		c.JSON(http.StatusOK, gin.H{"message": "Section deleted"})
	}
}

// ─── Field Handlers ───────────────────────────────────────────────────────────

// CreateFormField adds a field to a section
func CreateFormField(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubID(c)
		formID := c.Param("formId")
		sectionID := c.Param("sectionId")

		if !verifyFormOwnership(db, formID, clubID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
			return
		}

		var req struct {
			FieldType   string  `json:"field_type" binding:"required"`
			Label       string  `json:"label" binding:"required"`
			Placeholder *string `json:"placeholder"`
			HelperText  *string `json:"helper_text"`
			IsRequired  bool    `json:"is_required"`
			Options     *string `json:"options"`
			Validation  *string `json:"validation"`
			MapToField  *string `json:"map_to_field"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var maxOrder int
		_ = db.Get(&maxOrder, `SELECT COALESCE(MAX(order_index), -1) FROM club_form_fields WHERE section_id = ?`, sectionID)

		fieldID := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO club_form_fields 
			(uuid, section_id, form_id, field_type, label, placeholder, helper_text, is_required, options, validation, map_to_field, order_index)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, fieldID, sectionID, formID, req.FieldType, req.Label, req.Placeholder,
			req.HelperText, req.IsRequired, req.Options, req.Validation, req.MapToField, maxOrder+1)

		if err != nil {
			logrus.WithError(err).Error("[FORM] Failed to create field")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create field"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"id": fieldID, "message": "Field created"})
	}
}

// UpdateFormField updates a field's configuration
func UpdateFormField(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubID(c)
		formID := c.Param("formId")
		fieldID := c.Param("fieldId")

		if !verifyFormOwnership(db, formID, clubID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
			return
		}

		var req struct {
			Label       *string `json:"label"`
			Placeholder *string `json:"placeholder"`
			HelperText  *string `json:"helper_text"`
			IsRequired  *bool   `json:"is_required"`
			Options     *string `json:"options"`
			Validation  *string `json:"validation"`
			MapToField  *string `json:"map_to_field"`
			OrderIndex  *int    `json:"order_index"`
			SectionID   *string `json:"section_id"`
		}
		c.ShouldBindJSON(&req)

		_, err := db.Exec(`
			UPDATE club_form_fields
			SET label       = COALESCE(?, label),
			    placeholder = COALESCE(?, placeholder),
			    helper_text = COALESCE(?, helper_text),
			    is_required = COALESCE(?, is_required),
			    options     = COALESCE(?, options),
			    validation  = COALESCE(?, validation),
			    map_to_field = COALESCE(?, map_to_field),
			    order_index = COALESCE(?, order_index),
			    section_id  = COALESCE(?, section_id),
			    updated_at  = NOW()
			WHERE uuid = ? AND form_id = ?
		`, req.Label, req.Placeholder, req.HelperText, req.IsRequired,
			req.Options, req.Validation, req.MapToField, req.OrderIndex, req.SectionID,
			fieldID, formID)

		if err != nil {
			logrus.WithError(err).Error("[FORM] Failed to update field")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update field"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Field updated"})
	}
}

// DeleteFormField deletes a field
func DeleteFormField(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubID(c)
		formID := c.Param("formId")
		fieldID := c.Param("fieldId")

		if !verifyFormOwnership(db, formID, clubID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
			return
		}

		_, err := db.Exec(`DELETE FROM club_form_fields WHERE uuid = ? AND form_id = ?`, fieldID, formID)
		if err != nil {
			logrus.WithError(err).Error("[FORM] Failed to delete field")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete field"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Field deleted"})
	}
}

// ReorderFormItems reorders sections or fields using bulk order update
func ReorderFormItems(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubID(c)
		formID := c.Param("formId")

		if !verifyFormOwnership(db, formID, clubID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
			return
		}

		var req struct {
			Type  string `json:"type"`  // "sections" or "fields"
			Items []struct {
				UUID  string `json:"uuid"`
				Order int    `json:"order"`
			} `json:"items"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		table := "club_form_sections"
		if req.Type == "fields" {
			table = "club_form_fields"
		}

		for _, item := range req.Items {
			db.Exec(`UPDATE `+table+` SET order_index = ?, updated_at = NOW() WHERE uuid = ? AND form_id = ?`,
				item.Order, item.UUID, formID)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Order updated"})
	}
}

// ─── Private helpers ──────────────────────────────────────────────────────────

func verifyFormOwnership(db *sqlx.DB, formID, clubID string) bool {
	var exists bool
	db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM club_registration_forms WHERE uuid = ? AND club_id = ?)`, formID, clubID)
	return exists
}

func buildFormWithSections(db *sqlx.DB, form RegistrationForm) (*FormWithSections, error) {
	var sections []FormSection
	err := db.Select(&sections, `
		SELECT * FROM club_form_sections WHERE form_id = ? ORDER BY order_index ASC
	`, form.UUID)
	if err != nil {
		return nil, err
	}

	for i := range sections {
		var fields []FormField
		_ = db.Select(&fields, `
			SELECT * FROM club_form_fields WHERE section_id = ? ORDER BY order_index ASC
		`, sections[i].UUID)
		if fields == nil {
			fields = []FormField{}
		}
		sections[i].Fields = fields
	}

	if sections == nil {
		sections = []FormSection{}
	}

	return &FormWithSections{
		RegistrationForm: form,
		Sections:         sections,
	}, nil
}
