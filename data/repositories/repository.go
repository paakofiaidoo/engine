package repositories

import (
	"github.com/paakofiaidoo/juki/engine/data/models"
	"gorm.io/gorm"
)

/* ============================================
*			Repositories
* ============================================*/

type Repository interface {
	//Create(page *models.Page) error
	//project
	CreateProject(project *models.Project) error
	DeleteProject(id int) error
	GetProject(id int) (*models.Project, error)
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