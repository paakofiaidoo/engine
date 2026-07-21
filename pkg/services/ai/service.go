package ai

import (
	"context"
	"fmt"

	"juki-engine/pkg/data/models"
	"juki-engine/pkg/data/repositories"
)

// AgentType mirrors the proto enum as a string for DB storage.
const (
	AgentCodePilot = "code_pilot"
	AgentPM        = "pm"
	AgentArchitect = "architect"
	AgentDesigner  = "designer"
	AgentBuilder   = "builder"
)

// Service is the engine-owned AI orchestration layer.
//
// Responsibilities:
//  1. Build enriched prompts (project brief + file tree + recent logs + history)
//  2. Route to the best available provider (token-aware auto-rotation)
//  3. Persist conversation + message history
//  4. Stream responses back to callers
type Service struct {
	repo      repositories.Repository
	providers map[string]Provider // keyed by provider name: "claude", "gemini", ...
}

func NewService(repo repositories.Repository) *Service {
	return &Service{
		repo: repo,
		providers: map[string]Provider{
			"claude": NewClaudeProvider(),
			"gemini": NewGeminiProvider(),
		},
	}
}

// RegisterProvider allows adding pluggable providers (OpenAI, local models, etc.)
func (s *Service) RegisterProvider(p Provider) {
	s.providers[p.Name()] = p
}

// ─── Context Enrichment ──────────────────────────────────────────────────────

// ProjectContext is everything the AI needs to know about the project.
type ProjectContext struct {
	Brief       string   // JSON-serialized project brief
	RecentLogs  []string // last N console log messages
	FileTree    string   // serialized file tree summary
	History     []Message
}

// BuildProjectContext assembles the system prompt enrichment for a project.
// Per the architecture decision: most recent 50 console logs are always
// included so the AI has live debugging context.
func (s *Service) BuildProjectContext(projectID string) ProjectContext {
	ctx := ProjectContext{}

	// Project brief (from Sprint Planning, Phase F)
	if brief, err := s.repo.GetLatestBrief(projectID); err == nil && brief != nil {
		ctx.Brief = brief.BriefJSON
	}

	// Recent console logs — last 50, used as live debugging context
	if logs, _, err := s.repo.GetConsoleLogs(projectID, "", 50, 0, ""); err == nil {
		for _, l := range logs {
			ctx.RecentLogs = append(ctx.RecentLogs, fmt.Sprintf("[%s] %s", l.Level, l.Message))
		}
	}

	return ctx
}

// buildSystemPrompt composes the final system prompt sent to the provider.
func buildSystemPrompt(agentType string, pc ProjectContext) string {
	prompt := agentPrompt(agentType)

	if pc.Brief != "" {
		prompt += "\n\n## Project Brief\n" + pc.Brief
	}

	if len(pc.RecentLogs) > 0 {
		prompt += "\n\n## Recent Console Logs (most recent 50)\n"
		for _, l := range pc.RecentLogs {
			prompt += l + "\n"
		}
	}

	return prompt
}

// agentPrompt returns the base system prompt per agent type.
// Phase F/G will move these into pkg/prompts/ as versioned templates.
func agentPrompt(agentType string) string {
	switch agentType {
	case AgentPM:
		return "You are the PM Agent for Juki Builder. You run Sprint Planning Meetings: " +
			"ask structured questions about users, auth, payments, data models, and integrations, " +
			"then produce a structured JSON project plan."
	case AgentArchitect:
		return "You are the Architect Agent for Juki Builder. Given a PM plan, decide the technical " +
			"stack, packages to install, and folder structure. Use the available engine tools."
	case AgentDesigner:
		return "You are the Designer Agent for Juki Builder. Given a PM plan and brand keywords, " +
			"produce a theme configuration (colors, fonts, spacing)."
	case AgentBuilder:
		return "You are a Builder Agent for Juki Builder. You create pages, components, and data " +
			"models by calling engine tools (create_page, create_component, install_package, etc)."
	default: // AgentCodePilot
		return "You are Code Pilot, the in-editor AI assistant for Juki Builder, a visual Next.js IDE. " +
			"Help the user build their app. Be concise and actionable."
	}
}

// ─── Provider Routing & Auto-Rotation ────────────────────────────────────────

// resolveProvider finds the best available provider+credentials for a project,
// auto-rotating past exhausted providers (per the architecture decision:
// "auto switch when token finishes on one, multiple tokens for every agent").
func (s *Service) resolveProvider(projectID string) (Provider, *models.AIProvider, error) {
	cfg, err := s.repo.GetActiveAIProvider(projectID)
	if err != nil || cfg == nil {
		return nil, nil, fmt.Errorf("no active AI provider configured for project %s: %w", projectID, err)
	}

	provider, ok := s.providers[cfg.Name]
	if !ok {
		return nil, nil, fmt.Errorf("provider %q is not registered in the engine", cfg.Name)
	}

	return provider, cfg, nil
}

// ─── Public API ──────────────────────────────────────────────────────────────

// SendResult is returned from Send (non-streamed).
type SendResult struct {
	ConversationID string
	Message        *models.AIMessage
}

// Send builds context, calls the resolved provider, persists both turns,
// and returns the assistant's message.
func (s *Service) Send(ctx context.Context, projectID, conversationID, agentType, userMessage string, includeContext bool) (*SendResult, error) {
	conv, err := s.resolveConversation(projectID, conversationID, agentType)
	if err != nil {
		return nil, err
	}

	provider, cfg, err := s.resolveProvider(projectID)
	if err != nil {
		return nil, err
	}

	history, _ := s.repo.GetMessages(conv.ID, 50)

	var system string
	if includeContext {
		system = buildSystemPrompt(agentType, s.BuildProjectContext(projectID))
	} else {
		system = agentPrompt(agentType)
	}

	msgs := historyToMessages(history)
	msgs = append(msgs, Message{Role: "user", Content: userMessage})

	result, err := provider.Complete(ctx, CompletionRequest{
		APIKey:   cfg.APIKey,
		Model:    cfg.ModelName,
		System:   system,
		Messages: msgs,
	})
	if err != nil {
		return nil, err
	}

	// Persist user message
	userMsg := &models.AIMessage{ConversationID: conv.ID, Role: "user", Content: userMessage}
	_ = s.repo.AddMessage(userMsg)

	// Persist assistant message
	asstMsg := &models.AIMessage{
		ConversationID: conv.ID,
		Role:           "assistant",
		Content:        result.Content,
		TokensUsed:     result.TokensUsed,
		ProviderName:   provider.Name(),
	}
	if err := s.repo.AddMessage(asstMsg); err != nil {
		return nil, err
	}

	// Token accounting — drives auto-rotation on next call
	_ = s.repo.IncrementTokensUsed(cfg.ID, result.TokensUsed)

	return &SendResult{ConversationID: conv.ID, Message: asstMsg}, nil
}

// StreamResult chunks are forwarded as they arrive; final chunk includes the persisted message.
func (s *Service) Stream(ctx context.Context, projectID, conversationID, agentType, userMessage string, includeContext bool) (<-chan StreamChunk, error) {
	conv, err := s.resolveConversation(projectID, conversationID, agentType)
	if err != nil {
		return nil, err
	}

	provider, cfg, err := s.resolveProvider(projectID)
	if err != nil {
		return nil, err
	}

	history, _ := s.repo.GetMessages(conv.ID, 50)

	var system string
	if includeContext {
		system = buildSystemPrompt(agentType, s.BuildProjectContext(projectID))
	} else {
		system = agentPrompt(agentType)
	}

	msgs := historyToMessages(history)
	msgs = append(msgs, Message{Role: "user", Content: userMessage})

	upstream, err := provider.Stream(ctx, CompletionRequest{
		APIKey:   cfg.APIKey,
		Model:    cfg.ModelName,
		System:   system,
		Messages: msgs,
	})
	if err != nil {
		return nil, err
	}

	// Persist the user message immediately
	_ = s.repo.AddMessage(&models.AIMessage{ConversationID: conv.ID, Role: "user", Content: userMessage})

	out := make(chan StreamChunk, 32)
	go func() {
		defer close(out)
		for chunk := range upstream {
			if chunk.Done {
				// Persist assistant message on completion
				asstMsg := &models.AIMessage{
					ConversationID: conv.ID,
					Role:           "assistant",
					Content:        chunk.FinalContent,
					TokensUsed:     chunk.TokensUsed,
					ProviderName:   provider.Name(),
				}
				_ = s.repo.AddMessage(asstMsg)
				_ = s.repo.IncrementTokensUsed(cfg.ID, chunk.TokensUsed)
			}
			select {
			case out <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

func (s *Service) resolveConversation(projectID, conversationID, agentType string) (*models.AIConversation, error) {
	if conversationID != "" {
		return s.repo.GetConversation(conversationID)
	}
	return s.repo.GetOrCreateConversation(projectID, agentType)
}

func historyToMessages(history []*models.AIMessage) []Message {
	var out []Message
	for _, m := range history {
		out = append(out, Message{Role: m.Role, Content: m.Content})
	}
	return out
}
