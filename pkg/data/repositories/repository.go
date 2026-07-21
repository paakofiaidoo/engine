package repositories

import (
	"juki-engine/pkg/data/models"

	"gorm.io/gorm"
)

/* ============================================
*			Repositories
* ============================================*/

type Repository interface {
	// Project
	CreateProject(project *models.Project) error
	GetProject(id string) (*models.Project, error)
	GetProjects() ([]*models.Project, error)
	GetMaxPort() (int, error)
	UpdateProject(project *models.Project) error
	DeleteProject(id string) error

	// Page & Layout
	CreatePage(page *models.Page) error
	GetPage(id string) (*models.Page, error)
	UpdatePage(page *models.Page) error
	CreateLayout(layout *models.Layout) error
	UpdateLayout(layout *models.Layout) error

	// Activity & Terminal
	CreateActivityLog(log *models.ActivityLog) error
	CreateTerminalActivity(activity *models.TerminalActivity) error
	UpdateTerminalActivity(activity *models.TerminalActivity) error
	GetActiveTerminalActivity(projectID string) (*models.TerminalActivity, error)

	// Content
	SaveTree(projectID, pageID string, nodes []models.Content) error
	GetTree(pageID string) ([]models.Content, error)

	// Components
	CreateComponent(component *models.UserComponent) error
	UpdateComponent(component *models.UserComponent) error
	DeleteComponent(id string) error
	GetComponent(id string) (*models.UserComponent, error)
	ListComponents(projectID string) ([]*models.UserComponent, error)

	// Console Logs
	CreateConsoleLog(entry *models.ConsoleLog) error
	GetConsoleLogs(projectID, sessionID string, limit, offset int, level string) ([]*models.ConsoleLog, int64, error)
	ClearConsoleLogs(projectID, sessionID string) (int64, error)

	// AI Providers
	CreateAIProvider(p *models.AIProvider) error
	UpdateAIProvider(p *models.AIProvider) error
	ListAIProviders(projectID string) ([]*models.AIProvider, error)
	GetActiveAIProvider(projectID string) (*models.AIProvider, error)
	IncrementTokensUsed(providerID string, tokens int64) error

	// AI Conversations
	CreateConversation(c *models.AIConversation) error
	GetConversation(id string) (*models.AIConversation, error)
	GetOrCreateConversation(projectID, agentType string) (*models.AIConversation, error)
	AddMessage(msg *models.AIMessage) error
	GetMessages(conversationID string, limit int) ([]*models.AIMessage, error)

	// Project Briefs
	SaveProjectBrief(brief *models.ProjectBrief) error
	GetLatestBrief(projectID string) (*models.ProjectBrief, error)

	// Swarm Sessions
	CreateSwarmSession(s *models.SwarmSession) error
	UpdateSwarmSession(s *models.SwarmSession) error
	GetSwarmSession(id string) (*models.SwarmSession, error)
	GetActiveSwarmSession(projectID string) (*models.SwarmSession, error)

	// Marketplace Installs
	CreateMarketplaceInstall(i *models.MarketplaceInstall) error
	ListMarketplaceInstalls(projectID string) ([]*models.MarketplaceInstall, error)
	GetMarketplaceInstall(projectID, entryID string) (*models.MarketplaceInstall, error)
}

type repository struct {
	store *gorm.DB
}

/* ============================================
*			Repository Constructors
* ============================================*/

func NewRepository(db *gorm.DB) Repository {
	// We can embed the content logic directly or composing it.
	// Since repository struct is simple wrapper, let's just implement the methods on it via composition
	// or just add them to the struct methods in a new file.
	// But since Go doesn't dynamic mixin, we'll just implement them on *repository in content.go (by changing receiver).
	// Wait, I made contentRepository a separate struct.
	// Let's merged them or just add the fields.
	return &repository{store: db}
}
