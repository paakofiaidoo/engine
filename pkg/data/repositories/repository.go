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
}

type repository struct {
	store *gorm.DB
}

/* ============================================
*			Repository Constructors
* ============================================*/

func NewRepository(db *gorm.DB) Repository {
	return &repository{store: db}
}