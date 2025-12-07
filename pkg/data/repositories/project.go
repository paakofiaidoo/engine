package repositories

import (
	"juki-engine/pkg/data/models"
)

func (r *repository) GetProject(id string) (*models.Project, error) {
	project := &models.Project{}
	err := r.store.Preload("Pages").Preload("Layouts").First(project, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return project, nil
}

func (r *repository) GetProjects() ([]*models.Project, error) {
	projects := make([]*models.Project, 0)
	err := r.store.Preload("Pages").Find(&projects).Error
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func (r *repository) CreateProject(project *models.Project) error {
	return r.store.Create(project).Error
}

func (r *repository) UpdateProject(project *models.Project) error {
	return r.store.Save(project).Error
}

func (r *repository) CreatePage(page *models.Page) error {
	return r.store.Create(page).Error
}

func (r *repository) CreateLayout(layout *models.Layout) error {
	return r.store.Create(layout).Error
}

func (r *repository) DeleteProject(id string) error {
	return r.store.Delete(&models.Project{}, "id = ?", id).Error
}

func (r *repository) GetPage(id string) (*models.Page, error) {
	page := &models.Page{}
	err := r.store.First(page, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return page, nil
}

func (r *repository) UpdatePage(page *models.Page) error {
	return r.store.Save(page).Error
}

func (r *repository) GetMaxPort() (int, error) {
	var maxPort int
	// Select max(port) from projects. If null (no projects), it returns 0 (scan default for int)
	row := r.store.Model(&models.Project{}).Select("MAX(port)").Row()
	err := row.Scan(&maxPort)
	if err != nil {
		// If error is "NULL", it means no rows, which is fine, maxPort stays 0
		return 0, nil
	}
	return maxPort, nil
}