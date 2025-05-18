package models

import (
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

/* ============================================
*			Models
* ============================================*/

type Page struct {
	gorm.Model
	ID          string `gorm:"primaryKey"`
	Name        string
	Route       string
	CustomTheme bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Project struct {
	gorm.Model
	Name        string
	Description string
	Framework   string
}

/* ============================================
*			Model Methods
* ============================================*/

func (t *Page) BeforeCreate(*gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}

	return nil
}

func (t *Page) String() string {
	jsonBytes, _ := json.Marshal(t)
	return string(jsonBytes)
}