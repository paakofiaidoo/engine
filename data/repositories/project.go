package repositories

import "juki-engine/data/models"

func (r *repository) GetProject(id string) (*models.Project, error) {
	project := &models.Project{}
	err := r.store.Preload("Pages").First(project, "id = ?", id).Error
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
