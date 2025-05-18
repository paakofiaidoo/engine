package repositories

import "github.com/paakofiaidoo/juki/engine/data/models"

func (r *repository) GetProject(id int) (*models.Project, error) {
	project := &models.Project{}
	err := r.store.Find(project, id).Error
	if err != nil {
		return nil, err
	}
	return project, nil
}

func (r *repository) CreateProject(project *models.Project) error {
	return r.store.Create(project).Error
}

func (r *repository) UpdateProject(project *models.Project) error {
	return r.store.Save(project).Error
}

func (r *repository) DeleteProject(id int) error {
	return r.store.Delete(id).Error
}

func (r *repository) GetProjects() ([]*models.Project, error) {
	projects := make([]*models.Project, 0)
	err := r.store.Find(&projects).Error
	if err != nil {
		return nil, err
	}
	return projects, nil
}