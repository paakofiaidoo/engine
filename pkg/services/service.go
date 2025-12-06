package services

import (
	"juki-engine/data/dtos"
	"juki-engine/data/models"
	"juki-engine/data/repositories"
	"juki-engine/pkg/bridge"
	"juki-engine/pkg/scripts"
	"juki-engine/pkg/watcher"

	"github.com/fsnotify/fsnotify"
)

/* ============================================
*			Services
* ============================================*/

type Service interface {
	CreateProject(project dtos.Project) (*models.Project, error)
	GetProject(id string) (*models.Project, error)
	ListProjects() ([]dtos.Project, error)
	UpdateProject(project dtos.Project) error
	DeleteProject(id string) error
	SyncProject(id string) (*dtos.Project, error)
	PingWorker() (string, error)

	// Page Service
	CreatePage(projectID, name, route string) (*models.Page, error)
	GetPage(id string) (*models.Page, error)
	SavePage(projectID, pageID, content string) error

	// Build Service
	BuildProject(projectID string) (string, error)

	InstallPlugin(projectID, pluginName, version string) error
	ConfigureCMS(projectID, cmsType, configJSON string) error

	// Swarm Service
	RunSwarm(projectID, prompt string) (<-chan dtos.SwarmEvent, error)

	// File Watcher Service
	SubscribeToFileEvents(projectID string) (<-chan fsnotify.Event, error)
}

type service struct {
	repository repositories.Repository
	scripts    scripts.Scripts
	bridge     *bridge.Bridge
	watcher    *watcher.Watcher
}

/* ============================================
*			Service Constructors
* ============================================*/

func NewService(repository repositories.Repository, script scripts.Scripts, bridge *bridge.Bridge, watcher *watcher.Watcher) Service {
	return &service{
		repository: repository,
		scripts:    script,
		bridge:     bridge,
		watcher:    watcher,
	}
}
