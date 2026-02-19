package repositories

import (
	"juki-engine/pkg/data/models"

	"gorm.io/gorm"
)

/* ============================================
*			Repositories
* ============================================*/

type Repository interface {
	// Project
	CreateProject(project *models.Project) error
	GetProject(id string) (*models.Project, error)
	GetProjects() ([]*models.Project, error)
	GetMaxPort() (int, error)
	UpdateProject(project *models.Project) error
	DeleteProject(id string) error

	// Page & Layout
	CreatePage(page *models.Page) error
	GetPage(id string) (*models.Page, error)
	UpdatePage(page *models.Page) error
	CreateLayout(layout *models.Layout) error
	UpdateLayout(layout *models.Layout) error

	// Activity & Terminal
	CreateActivityLog(log *models.ActivityLog) error
	CreateTerminalActivity(activity *models.TerminalActivity) error
	UpdateTerminalActivity(activity *models.TerminalActivity) error
	GetActiveTerminalActivity(projectID string) (*models.TerminalActivity, error)

	// Content
	SaveTree(projectID, pageID string, nodes []models.Content) error
	GetTree(pageID string) ([]models.Content, error)
}

type repository struct {
	store *gorm.DB
}

/* ============================================
*			Repository Constructors
* ============================================*/

func NewRepository(db *gorm.DB) Repository {
	// We can embed the content logic directly or composing it.
	// Since repository struct is simple wrapper, let's just implement the methods on it via composition
	// or just add them to the struct methods in a new file.
	// But since Go doesn't dynamic mixin, we'll just implement them on *repository in content.go (by changing receiver).
	// Wait, I made contentRepository a separate struct.
	// Let's merged them or just add the fields.
	return &repository{store: db}
}
