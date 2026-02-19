package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

type Content struct {
	ID        string         `gorm:"primaryKey;type:text" json:"id"`
	ProjectID string         `gorm:"index;type:text;not null" json:"project_id"`
	PageID    *string        `gorm:"index;type:text" json:"page_id,omitempty"`   // Nullable for shared/orphan content
	LayoutID  *string        `gorm:"index;type:text" json:"layout_id,omitempty"` // Nullable
	ParentID  *string        `gorm:"index;type:text" json:"parent_id,omitempty"` // Nullable for root nodes
	Type      string         `gorm:"type:text;not null" json:"type"`             // ELEMENT, COMPONENT, TEXT
	Tag       string         `gorm:"type:text" json:"tag"`                       // div, h1, Button
	Props     JSONMap        `gorm:"type:text" json:"props"`                     // JSON string or map
	Text      string         `gorm:"type:text" json:"text"`                      // For Text nodes
	Order     int            `gorm:"type:integer;default:0" json:"order"`        // Sibling order
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	Children []Content `gorm:"foreignKey:ParentID;references:ID" json:"children,omitempty"`
}

// JSONMap helper for GORM
type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		// handle string if driver returns string
		str, ok := value.(string)
		if ok {
			bytes = []byte(str)
		} else {
			return errors.New("type assertion to []byte failed")
		}
	}
	return json.Unmarshal(bytes, j)
}
