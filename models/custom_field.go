package models

import (
	"encoding/json"
	"time"
)

// TournamentCustomField represents a custom registration field or visual layout block defined by an organizer
type TournamentCustomField struct {
	ID                   int64           `json:"id" db:"id"`
	UUID                 string          `json:"uuid" db:"uuid"`
	TournamentUUID       string          `json:"tournament_uuid" db:"tournament_uuid"`
	FieldKey             string          `json:"field_key" db:"field_key"`
	LabelID              string          `json:"label_id" db:"label_id"`
	LabelEN              *string         `json:"label_en,omitempty" db:"label_en"`
	PlaceholderID        *string         `json:"placeholder_id,omitempty" db:"placeholder_id"`
	PlaceholderEN        *string         `json:"placeholder_en,omitempty" db:"placeholder_en"`
	DescriptionID        *string         `json:"description_id,omitempty" db:"description_id"`
	DescriptionEN        *string         `json:"description_en,omitempty" db:"description_en"`
	FieldType            string          `json:"field_type" db:"field_type"` // text, textarea, number, select, radio, checkbox, date, file
	OptionsJSON          *string         `json:"-" db:"options_json"`
	Options              []string        `json:"options"`
	IsRequired           bool            `json:"is_required" db:"is_required"`
	IsActive             bool            `json:"is_active" db:"is_active"`
	AppliesToCategoryIDs *string         `json:"-" db:"applies_to_category_ids"`
	CategoryIDs          []string        `json:"applies_to_category_ids"`
	DisplayOrder         int             `json:"display_order" db:"display_order"`
	ColSpan              int             `json:"col_span" db:"col_span"`                     // 12 (100%), 6 (50%), 4 (33%), 3 (25%), 8 (66%)
	ElementType          string          `json:"element_type" db:"element_type"`             // field, heading, divider, spacer, notice
	StyleConfigJSON      *string         `json:"-" db:"style_config"`
	StyleConfig          map[string]any  `json:"style_config,omitempty"`
	CreatedAt            time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at" db:"updated_at"`
}

// UnmarshalJSONCustom populates parsed Options, CategoryIDs, and StyleConfig from raw JSON columns
func (f *TournamentCustomField) ParseJSONFields() {
	if f.OptionsJSON != nil && *f.OptionsJSON != "" {
		var opts []string
		if err := json.Unmarshal([]byte(*f.OptionsJSON), &opts); err == nil {
			f.Options = opts
		} else {
			f.Options = []string{}
		}
	} else {
		f.Options = []string{}
	}

	if f.AppliesToCategoryIDs != nil && *f.AppliesToCategoryIDs != "" {
		var cats []string
		if err := json.Unmarshal([]byte(*f.AppliesToCategoryIDs), &cats); err == nil {
			f.CategoryIDs = cats
		} else {
			f.CategoryIDs = []string{}
		}
	} else {
		f.CategoryIDs = []string{}
	}

	if f.StyleConfigJSON != nil && *f.StyleConfigJSON != "" {
		var styles map[string]any
		if err := json.Unmarshal([]byte(*f.StyleConfigJSON), &styles); err == nil {
			f.StyleConfig = styles
		} else {
			f.StyleConfig = make(map[string]any)
		}
	} else {
		f.StyleConfig = make(map[string]any)
	}

	if f.ColSpan <= 0 {
		f.ColSpan = 12
	}
	if f.ElementType == "" {
		f.ElementType = "field"
	}
}

// CustomFieldInput represents the payload for creating or updating a custom field or layout block
type CustomFieldInput struct {
	FieldKey             string         `json:"field_key"`
	LabelID              string         `json:"label_id"`
	LabelEN              *string        `json:"label_en"`
	PlaceholderID        *string        `json:"placeholder_id"`
	PlaceholderEN        *string        `json:"placeholder_en"`
	DescriptionID        *string        `json:"description_id"`
	DescriptionEN        *string        `json:"description_en"`
	FieldType            string         `json:"field_type"`
	Options              []string       `json:"options"`
	IsRequired           bool           `json:"is_required"`
	IsActive             *bool          `json:"is_active"`
	AppliesToCategoryIDs []string       `json:"applies_to_category_ids"`
	DisplayOrder         *int           `json:"display_order"`
	ColSpan              *int           `json:"col_span"`
	ElementType          *string        `json:"element_type"`
	StyleConfig          map[string]any `json:"style_config"`
}

// CustomFieldReorderItem represents an item in the reorder list
type CustomFieldReorderItem struct {
	FieldUUID    string `json:"field_uuid" binding:"required"`
	DisplayOrder int    `json:"display_order" binding:"required"`
}

// CustomFieldReorderRequest represents the payload to reorder fields
type CustomFieldReorderRequest struct {
	Orders []CustomFieldReorderItem `json:"orders" binding:"required"`
}

// ParticipantCustomFieldValue represents an athlete's answer to a custom field
type ParticipantCustomFieldValue struct {
	ID              int64     `json:"id" db:"id"`
	UUID            string    `json:"uuid" db:"uuid"`
	ParticipantUUID string    `json:"participant_uuid" db:"participant_uuid"`
	FieldUUID       string    `json:"field_uuid" db:"field_uuid"`
	FieldValue      *string   `json:"field_value" db:"field_value"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// CustomFieldValueDetail represents a joined custom field definition and its submitted answer
type CustomFieldValueDetail struct {
	FieldUUID    string   `json:"field_uuid" db:"field_uuid"`
	FieldKey     string   `json:"field_key" db:"field_key"`
	LabelID      string   `json:"label_id" db:"label_id"`
	LabelEN      *string  `json:"label_en,omitempty" db:"label_en"`
	FieldType    string   `json:"field_type" db:"field_type"`
	FieldValue   *string  `json:"field_value" db:"field_value"`
	Options      []string `json:"options,omitempty"`
	IsRequired   bool     `json:"is_required" db:"is_required"`
	DisplayOrder int      `json:"display_order" db:"display_order"`
	ColSpan      int      `json:"col_span" db:"col_span"`
	ElementType  string   `json:"element_type" db:"element_type"`
}
