package repositories

import "juki-engine/pkg/data/models"

func (r *repository) CreateComponent(component *models.UserComponent) error {
	return r.store.Create(component).Error
}

func (r *repository) UpdateComponent(component *models.UserComponent) error {
	return r.store.Save(component).Error
}

func (r *repository) DeleteComponent(id string) error {
	return r.store.Delete(&models.UserComponent{}, "id = ?", id).Error
}

func (r *repository) GetComponent(id string) (*models.UserComponent, error) {
	component := &models.UserComponent{}
	err := r.store.First(component, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return component, nil
}

func (r *repository) ListComponents(projectID string) ([]*models.UserComponent, error) {
	var components []*models.UserComponent
	err := r.store.Where("project_id = ?", projectID).Find(&components).Error
	if err != nil {
		return nil, err
	}
	return components, nil
}
