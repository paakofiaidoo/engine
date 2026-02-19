package repositories

import (
	"juki-engine/pkg/data/models"
)

func (r *repository) CreateActivityLog(log *models.ActivityLog) error {
	return r.store.Create(log).Error
}

func (r *repository) CreateTerminalActivity(activity *models.TerminalActivity) error {
	return r.store.Create(activity).Error
}

func (r *repository) UpdateTerminalActivity(activity *models.TerminalActivity) error {
	return r.store.Save(activity).Error
}

func (r *repository) GetActiveTerminalActivity(projectID string) (*models.TerminalActivity, error) {
	activity := &models.TerminalActivity{}
	// Status = "RUNNING"
	err := r.store.Where("project_id = ? AND status = ?", projectID, "RUNNING").First(activity).Error
	if err != nil {
		return nil, err
	}
	return activity, nil
}
