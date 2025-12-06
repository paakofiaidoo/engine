package repositories

import (
	"juki-engine/data/models"

	"gorm.io/gorm"
)

/* ============================================
*			Repositories
* ============================================*/

type Repository interface {
	CreatePage(page *models.Page) error
	GetPage(id string) (*models.Page, error)
	UpdatePage(page *models.Page) error
	CreateProject(project *models.Project) error
	DeleteProject(id string) error
	GetProject(id string) (*models.Project, error)
	GetProjects() ([]*models.Project, error)
	UpdateProject(project *models.Project) error
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
