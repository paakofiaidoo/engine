package models

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ============================================
*			Models
* ============================================*/

type Page struct {
	gorm.Model
	ID              string `gorm:"primaryKey"`
	ProjectID       string `gorm:"index"`
	Name            string
	Description     string
	Route           string
	Content         string // JSON string of canvas items
	ComposedContent string // JSON string of layout + page items
	RawContent      string // Raw file content (TSX)
	CustomTheme     bool
}

type Layout struct {
	gorm.Model
	ID             string `gorm:"primaryKey"`
	ProjectID      string `gorm:"index"`
	Name           string
	Route          string // The directory route (e.g. "/" or "/blog")
	Content        string // JSON
	RawContent     string // File content
	IsRoot         bool
	ParentLayoutID *string
}

type Project struct {
	gorm.Model
	ID          string `gorm:"primaryKey"`
	Name        string
	Path        string
	Description string
	Framework   string
	ApiKey      string // Encrypted or plain for now (MVP: plain)
	Port        int
	Pages       []Page   `gorm:"foreignKey:ProjectID"`
	Layouts     []Layout `gorm:"foreignKey:ProjectID"`
}

type UserComponent struct {
	gorm.Model
	ID        string `gorm:"primaryKey"`
	ProjectID string `gorm:"index"`
	Name      string
	Content   string // JSON stringified AnyCanvasItem tree
}

func (t *UserComponent) BeforeCreate(*gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	return nil
}

// ConsoleLog stores a single browser console.log/warn/error/info entry.
// Retention: rolling 1000 per project (enforced in repository).
type ConsoleLog struct {
	gorm.Model
	ID             string `gorm:"primaryKey"`
	ProjectID      string `gorm:"index"`
	SessionID      string `gorm:"index"`
	Level          string // "log" | "info" | "warn" | "error" | "debug"
	Message        string
	ArgsJSON       string // JSON array of serialized args
	TimestampMs    int64
	SourceLocation string // "file.tsx:12:4" from Error.stack
}

func (t *ConsoleLog) BeforeCreate(*gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	return nil
}

// AIProvider stores user-configured AI provider credentials per project.
type AIProvider struct {
	gorm.Model
	ID           string `gorm:"primaryKey"`
	ProjectID    string `gorm:"index"`
	Name         string // "claude" | "gemini" | "openai"
	APIKey       string // encrypted in production
	ModelName    string // e.g. "claude-opus-4-5" or "gemini-2.0-flash"
	TokenLimit   int64
	TokensUsed   int64
	IsActive     bool
	Priority     int // lower = higher priority for auto-rotation
}

func (t *AIProvider) BeforeCreate(*gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	return nil
}

// AIConversation groups messages into a thread (per project + agent type).
type AIConversation struct {
	gorm.Model
	ID        string `gorm:"primaryKey"`
	ProjectID string `gorm:"index"`
	AgentType string // "code_pilot" | "pm" | "architect" | "designer" | "builder"
	Messages  []AIMessage `gorm:"foreignKey:ConversationID"`
}

func (t *AIConversation) BeforeCreate(*gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	return nil
}

// AIMessage is one turn in a conversation.
type AIMessage struct {
	gorm.Model
	ID             string `gorm:"primaryKey"`
	ConversationID string `gorm:"index"`
	Role           string // "user" | "assistant" | "system"
	Content        string
	TokensUsed     int64
	ProviderName   string // which provider generated this response
}

func (t *AIMessage) BeforeCreate(*gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	return nil
}

// ProjectBrief stores the approved sprint planning output for a project.
type ProjectBrief struct {
	gorm.Model
	ID        string `gorm:"primaryKey"`
	ProjectID string `gorm:"index"`
	BriefJSON string // JSON serialized ProjectBriefData
	Version   int
}

func (t *ProjectBrief) BeforeCreate(*gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	return nil
}

// SwarmSession tracks a multi-agent execution run.
type SwarmSession struct {
	gorm.Model
	ID        string `gorm:"primaryKey"`
	ProjectID string `gorm:"index"`
	BriefID   string
	Status    string // "idle" | "planning" | "reviewing" | "executing" | "done" | "cancelled"
	PlanJSON  string // current execution plan
}

func (t *SwarmSession) BeforeCreate(*gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	return nil
}

// MarketplaceInstall records that a registry entry has been installed into a project
// (copy / npm / patch). Used to power "Installed" views and avoid duplicate installs.
type MarketplaceInstall struct {
	gorm.Model
	ID           string `gorm:"primaryKey"`
	ProjectID    string `gorm:"index"`
	EntryID      string `gorm:"index"`
	Name         string
	Type         string // "template" | "component" | "plugin" | "icon_pack"
	InstalledRef string // component id / page route / package name
}

func (t *MarketplaceInstall) BeforeCreate(*gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	return nil
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

func (t *Layout) BeforeCreate(*gorm.DB) error {
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
