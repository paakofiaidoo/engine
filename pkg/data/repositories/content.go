package repositories

import (
	"juki-engine/pkg/data/models"

	"gorm.io/gorm"
)

type ContentRepository interface {
	// SaveTree replaces the content tree for a page (transactional delete old + insert new)
	SaveTree(projectID, pageID string, nodes []models.Content) error
	// GetTree fetches all content nodes for a page
	GetTree(pageID string) ([]models.Content, error)
}

// SaveTree performs a full replace of the content tree for a given page.
// This is a simplified approach: we delete all existing content for the page and insert the new tree.
// In a production scenario with collaborative editing, we would want fine-grained patches.
func (r *repository) SaveTree(projectID, pageID string, nodes []models.Content) error {
	return r.store.Transaction(func(tx *gorm.DB) error {
		// 1. Delete existing content for this page
		// Hard delete or soft delete? Let's do hard delete for now to keep the table clean while iterating.
		// If we use SoftDelete in the model, Unscoped() forces hard delete.
		if err := tx.Unscoped().Where("project_id = ? AND page_id = ?", projectID, pageID).Delete(&models.Content{}).Error; err != nil {
			return err
		}

		// 2. Insert new nodes
		if len(nodes) > 0 {
			if err := tx.Create(&nodes).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *repository) GetTree(pageID string) ([]models.Content, error) {
	var nodes []models.Content
	// Order by nesting level (if possible) or just fetch all and reconstruct in memory.
	// Ordering by `created_at` or `order` is important for siblings.
	if err := r.store.Where("page_id = ?", pageID).Order("\"order\" ASC").Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}
