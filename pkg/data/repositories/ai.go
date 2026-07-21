package repositories

import (
	"juki-engine/pkg/data/models"

	"gorm.io/gorm"
)

// ─── AI Providers ────────────────────────────────────────────────────────────

func (r *repository) CreateAIProvider(p *models.AIProvider) error {
	return r.store.Create(p).Error
}

func (r *repository) UpdateAIProvider(p *models.AIProvider) error {
	return r.store.Save(p).Error
}

func (r *repository) ListAIProviders(projectID string) ([]*models.AIProvider, error) {
	var providers []*models.AIProvider
	err := r.store.Where("project_id = ?", projectID).Order("priority ASC").Find(&providers).Error
	return providers, err
}

// GetActiveAIProvider returns the lowest-priority provider that still has tokens remaining.
func (r *repository) GetActiveAIProvider(projectID string) (*models.AIProvider, error) {
	var provider models.AIProvider
	err := r.store.
		Where("project_id = ? AND is_active = ? AND (token_limit = 0 OR tokens_used < token_limit)", projectID, true).
		Order("priority ASC").
		First(&provider).Error
	return &provider, err
}

func (r *repository) IncrementTokensUsed(providerID string, tokens int64) error {
	return r.store.Model(&models.AIProvider{}).
		Where("id = ?", providerID).
		UpdateColumn("tokens_used", gorm.Expr("tokens_used + ?", tokens)).
		Error
}

// ─── AI Conversations ────────────────────────────────────────────────────────

func (r *repository) CreateConversation(c *models.AIConversation) error {
	return r.store.Create(c).Error
}

func (r *repository) GetConversation(id string) (*models.AIConversation, error) {
	var c models.AIConversation
	err := r.store.Preload("Messages").First(&c, "id = ?", id).Error
	return &c, err
}

// GetOrCreateConversation returns an existing conversation for (project, agentType) or creates a new one.
func (r *repository) GetOrCreateConversation(projectID, agentType string) (*models.AIConversation, error) {
	var c models.AIConversation
	err := r.store.Where("project_id = ? AND agent_type = ?", projectID, agentType).
		Order("created_at DESC").
		First(&c).Error
	if err != nil {
		// Create new
		c = models.AIConversation{ProjectID: projectID, AgentType: agentType}
		if err2 := r.store.Create(&c).Error; err2 != nil {
			return nil, err2
		}
	}
	return &c, nil
}

func (r *repository) AddMessage(msg *models.AIMessage) error {
	return r.store.Create(msg).Error
}

func (r *repository) GetMessages(conversationID string, limit int) ([]*models.AIMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	var msgs []*models.AIMessage
	err := r.store.Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Limit(limit).
		Find(&msgs).Error
	return msgs, err
}

// ─── Project Briefs ──────────────────────────────────────────────────────────

func (r *repository) SaveProjectBrief(brief *models.ProjectBrief) error {
	return r.store.Save(brief).Error
}

func (r *repository) GetLatestBrief(projectID string) (*models.ProjectBrief, error) {
	var brief models.ProjectBrief
	err := r.store.Where("project_id = ?", projectID).
		Order("version DESC").
		First(&brief).Error
	return &brief, err
}

// ─── Swarm Sessions ──────────────────────────────────────────────────────────

func (r *repository) CreateSwarmSession(s *models.SwarmSession) error {
	return r.store.Create(s).Error
}

func (r *repository) UpdateSwarmSession(s *models.SwarmSession) error {
	return r.store.Save(s).Error
}

func (r *repository) GetSwarmSession(id string) (*models.SwarmSession, error) {
	var s models.SwarmSession
	err := r.store.First(&s, "id = ?", id).Error
	return &s, err
}

func (r *repository) GetActiveSwarmSession(projectID string) (*models.SwarmSession, error) {
	var s models.SwarmSession
	err := r.store.Where("project_id = ? AND status NOT IN ('done','cancelled')", projectID).
		Order("created_at DESC").
		First(&s).Error
	return &s, err
}
