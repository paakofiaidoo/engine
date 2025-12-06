package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ============================================
*			Models
* ============================================*/

type Page struct {
	gorm.Model
	ID          string `gorm:"primaryKey"`
	ProjectID   string `gorm:"index"`
	Name        string
	Description string
	Route       string
	Content     string // JSON string of canvas items
	RawContent  string // Raw file content (TSX)
	CustomTheme bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Project struct {
	gorm.Model
	ID          string `gorm:"primaryKey"`
	Name        string
	Path        string
	Description string
	Framework   string
	ApiKey      string // Encrypted or plain for now (MVP: plain)
	Pages       []Page `gorm:"foreignKey:ProjectID"`
}

type File struct {
	gorm.Model
	ProjectID uint
	Path      string `gorm:"index"`
	Hash      string // For optimistic concurrency
}

type Symbol struct {
	gorm.Model
	FileID uint
	Name   string
	Kind   string // "Component", "Function", "Variable"
	Line   int
}

// Draft State for Command Protocol
type DraftCommand struct {
	gorm.Model
	SessionID string `gorm:"index"`
	Command   string // JSON string
	Status    string // "Pending", "Applied", "Failed"
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
func (t *Project) BeforeCreate(*gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}

	return nil
}

func (t *Page) String() string {
	jsonBytes, _ := json.Marshal(t)
	return string(jsonBytes)
}
