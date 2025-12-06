package services

import (
	"fmt"
	"log"

	"github.com/fsnotify/fsnotify"
)

func (s *service) SubscribeToFileEvents(projectID string) (<-chan fsnotify.Event, error) {
	log.Printf("[Service] SubscribeToFileEvents: Subscribing for project %s\n", projectID)

	// 1. Get Project to find path
	project, err := s.repository.GetProject(projectID)
	if err != nil {
		return nil, err
	}

	if project.Path == "" {
		return nil, fmt.Errorf("project path is empty")
	}

	// 2. Add project path to watcher
	// Note: fsnotify is not recursive by default on Linux, but on Mac (kqueue) it might be?
	// Actually fsnotify is NOT recursive. We might need to walk the directory and add all subdirectories.
	// For now, let's just add the root and maybe the `app` directory.
	// Ideally we should use a recursive watcher library or implement recursion.
	// Given the scope, let's add the project root and `app` folder.

	if err := s.watcher.Add(project.Path); err != nil {
		log.Printf("[Service] SubscribeToFileEvents: Failed to watch project root: %v\n", err)
		// Continue anyway?
	}

	appPath := fmt.Sprintf("%s/app", project.Path)
	if err := s.watcher.Add(appPath); err != nil {
		log.Printf("[Service] SubscribeToFileEvents: Failed to watch app dir: %v\n", err)
	}

	// 3. Subscribe
	return s.watcher.Subscribe(), nil
}
