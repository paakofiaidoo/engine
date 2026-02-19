package services

import (
	"encoding/json"
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
	page, err := s.repository.GetPage(id)
	if err != nil {
		return nil, err
	}

	// Reconstruct Content Tree from DB
	contentNodes, err := s.repository.GetTree(id)
	if err == nil && len(contentNodes) > 0 {
		// Build Tree
		nodeMap := make(map[string]interface{})
		var rootNodes []interface{}

		// Pass 1: Create Map
		for _, node := range contentNodes {
			// Convert node to map
			nodeMapRaw := map[string]interface{}{
				"id":   node.ID,
				"type": node.Type,
				"tag":  node.Tag,
				"text": node.Text,
			}
			if node.Props != nil {
				nodeMapRaw["props"] = node.Props
			}
			// Initialize content list (empty array)
			nodeMapRaw["content"] = []interface{}{}

			nodeMap[node.ID] = nodeMapRaw
		}

		// Pass 2: Link Children
		for _, node := range contentNodes {
			nodeObj := nodeMap[node.ID].(map[string]interface{})
			if node.ParentID != nil {
				if parentObj, ok := nodeMap[*node.ParentID].(map[string]interface{}); ok {
					children := parentObj["content"].([]interface{})
					children = append(children, nodeObj)
					parentObj["content"] = children
				}
			} else {
				// Is Root
				rootNodes = append(rootNodes, nodeObj)
			}
		}

		// Update Page Content
		if len(rootNodes) > 0 {
			// Use indentation for debugging visibility? No, compact is fine.
			jsonBytes, _ := json.Marshal(rootNodes)
			page.Content = string(jsonBytes)
		}
	}

	return page, nil
}
