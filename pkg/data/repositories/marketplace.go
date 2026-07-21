package repositories

import (
	"juki-engine/pkg/data/models"
)

func (r *repository) CreateMarketplaceInstall(i *models.MarketplaceInstall) error {
	return r.store.Create(i).Error
}

func (r *repository) ListMarketplaceInstalls(projectID string) ([]*models.MarketplaceInstall, error) {
	var installs []*models.MarketplaceInstall
	err := r.store.Where("project_id = ?", projectID).Order("created_at DESC").Find(&installs).Error
	return installs, err
}

func (r *repository) GetMarketplaceInstall(projectID, entryID string) (*models.MarketplaceInstall, error) {
	var install models.MarketplaceInstall
	err := r.store.Where("project_id = ? AND entry_id = ?", projectID, entryID).First(&install).Error
	if err != nil {
		return nil, err
	}
	return &install, nil
}
