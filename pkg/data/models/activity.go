package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActivityLog struct {
	gorm.Model
	ID         string `gorm:"primaryKey"`
	ProjectID  string `gorm:"index"`
	ActionType string // "MUTATION", "QUERY", "ERROR"
	Method     string // gRPC method name
	Payload    string // Request JSON
	Response   string // Response JSON
	StatusCode int
	DurationMs int64
	Timestamp  time.Time
}

type TerminalActivity struct {
	gorm.Model
	ID        string `gorm:"primaryKey"`
	ProjectID string `gorm:"index"`
	Status    string // "RUNNING", "STOPPED", "FAILED", "KILLED"
	Type      string // "DEV", "BUILD", "INSTALL", "CUSTOM"
	PID       int
	StartedAt time.Time
	StoppedAt *time.Time
}

func (t *ActivityLog) BeforeCreate(*gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	return nil
}

func (t *TerminalActivity) BeforeCreate(*gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	if t.StartedAt.IsZero() {
		t.StartedAt = time.Now()
	}
	return nil
}
