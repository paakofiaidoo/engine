package services

import (
	"juki-engine/pkg/data/models"

	"github.com/google/uuid"
)

func (s *service) CreatePage(projectID, name, route string) (*models.Page, error) {
	page := &models.Page{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		Name:      name,
		Route:     route,
		Content:   "", // Empty initially
	}

	if err := s.repository.CreatePage(page); err != nil {
		return nil, err
	}

	return page, nil
}

func (s *service) GetPage(id string) (*models.Page, error) {
	return s.repository.GetPage(id)
}