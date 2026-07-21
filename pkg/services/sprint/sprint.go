package sprint

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"juki-engine/pkg/data/models"
	"juki-engine/pkg/data/repositories"
	"juki-engine/pkg/services/ai"
)

// BriefData is the structured output of a Sprint Planning Meeting.
// Mirrors the JSON shape agreed in the architecture plan:
//
//	{
//	  "description": "...",
//	  "users": ["buyer", "seller"],
//	  "pages": ["home", "listings", ...],
//	  "plugins": ["nextauth", "stripe", ...],
//	  "dataModels": ["Product", "User", ...],
//	  "theme": { "style": "minimal", "primaryColor": "amber" }
//	}
type BriefData struct {
	Description string   `json:"description"`
	Users       []string `json:"users"`
	Pages       []string `json:"pages"`
	Plugins     []string `json:"plugins"`
	DataModels  []string `json:"dataModels"`
	Theme       struct {
		Style        string `json:"style"`
		PrimaryColor string `json:"primaryColor"`
	} `json:"theme"`
}

// Service runs Sprint Planning Meetings — guided PM Agent Q&A sessions
// that produce a structured ProjectBrief before any code is generated.
type Service struct {
	repo  repositories.Repository
	aiSvc *ai.Service
}

func NewService(repo repositories.Repository, aiSvc *ai.Service) *Service {
	return &Service{repo: repo, aiSvc: aiSvc}
}

// pmSystemPrompt is the structured Q&A guide for the PM Agent.
// In Phase G this moves to pkg/prompts/ as a versioned template.
const pmSystemPrompt = `You are the PM Agent running a "Sprint Planning Meeting" for Juki Builder.
Your job: deeply understand the user's project idea through structured conversation
BEFORE any code gets written.

Ask about, in order, one or two at a time (don't overwhelm):
1. Who are the users? (e.g. buyers, sellers, admins)
2. Authentication needs? (social login, email/password, none)
3. Payments? (which provider, one-time vs subscription)
4. Core data models? (e.g. Product, Order, User, Review)
5. Pages needed? (home, listing, detail, checkout, profile, etc.)
6. Integrations? (shipping, messaging, analytics, email)
7. Visual style / theme preferences? (minimal, bold, colorful; primary color)

Once you have enough information, output a final structured brief as JSON
wrapped in a fenced code block tagged "json", in EXACTLY this shape:

` + "```json" + `
{
  "description": "...",
  "users": ["..."],
  "pages": ["..."],
  "plugins": ["..."],
  "dataModels": ["..."],
  "theme": { "style": "...", "primaryColor": "..." }
}
` + "```" + `

Keep responses short and conversational. Do not output the JSON brief until
you've gathered enough information from the user — usually after 3-5 exchanges.`

// StartSession begins a new PM conversation seeded with the user's one-line idea.
func (s *Service) StartSession(ctx context.Context, projectID, description string) (conversationID string, opening string, err error) {
	conv, err := s.repo.GetOrCreateConversation(projectID, ai.AgentPM)
	if err != nil {
		return "", "", err
	}

	userMsg := description
	if userMsg == "" {
		userMsg = "I'd like to plan a new project. Please ask me about it."
	}

	result, err := s.aiSvc.Send(ctx, projectID, conv.ID, ai.AgentPM, userMsg, false)
	if err != nil {
		// Graceful fallback — engine may have no AI provider configured yet.
		fallback := fallbackOpening(description)
		_ = s.repo.AddMessage(&models.AIMessage{ConversationID: conv.ID, Role: "user", Content: userMsg})
		_ = s.repo.AddMessage(&models.AIMessage{ConversationID: conv.ID, Role: "assistant", Content: fallback, ProviderName: "fallback"})
		return conv.ID, fallback, nil
	}

	return conv.ID, result.Message.Content, nil
}

func fallbackOpening(description string) string {
	base := "Let's plan your project! "
	if description != "" {
		base += fmt.Sprintf("You said: %q. ", description)
	}
	return base + "First — who are the main users of this app? (e.g. customers, admins, both?) " +
		"\n\n_(Note: no AI provider is configured yet — configure one in Settings to enable the full guided experience.)_"
}

// SendMessage continues the planning conversation. Returns the PM's reply
// and, if the reply contains a fenced ```json brief block, the parsed brief.
func (s *Service) SendMessage(ctx context.Context, projectID, conversationID, message string) (reply string, brief *BriefData, err error) {
	// Inject the PM system prompt context by relying on agent_type=pm
	// (ai.Service already attaches the PM base prompt via agentPrompt()).
	result, sendErr := s.aiSvc.Send(ctx, projectID, conversationID, ai.AgentPM, message, true)
	if sendErr != nil {
		return "", nil, sendErr
	}

	reply = result.Message.Content
	brief = extractBrief(reply)
	return reply, brief, nil
}

// extractBrief looks for a ```json ... ``` fenced block and parses it as BriefData.
func extractBrief(text string) *BriefData {
	const fence = "```json"
	start := strings.Index(text, fence)
	if start == -1 {
		return nil
	}
	rest := text[start+len(fence):]
	end := strings.Index(rest, "```")
	if end == -1 {
		return nil
	}
	jsonStr := strings.TrimSpace(rest[:end])

	var brief BriefData
	if err := json.Unmarshal([]byte(jsonStr), &brief); err != nil {
		return nil
	}
	return &brief
}

// ApproveBrief persists the brief as the canonical version for the project,
// incrementing the version number from any prior brief.
func (s *Service) ApproveBrief(projectID, briefJSON string) (*models.ProjectBrief, error) {
	version := 1
	if existing, err := s.repo.GetLatestBrief(projectID); err == nil && existing != nil {
		version = existing.Version + 1
	}

	brief := &models.ProjectBrief{
		ProjectID: projectID,
		BriefJSON: briefJSON,
		Version:   version,
	}

	if err := s.repo.SaveProjectBrief(brief); err != nil {
		return nil, err
	}

	return brief, nil
}

// GetBrief returns the latest approved brief for a project, if any.
func (s *Service) GetBrief(projectID string) (*models.ProjectBrief, bool) {
	brief, err := s.repo.GetLatestBrief(projectID)
	if err != nil || brief == nil || brief.ID == "" {
		return nil, false
	}
	return brief, true
}

// ToProtoData converts a BriefData (parsed from AI output) plus DB metadata
// into the wire format. Used when streaming a draft brief mid-conversation.
func (b *BriefData) ToJSON() string {
	data, _ := json.Marshal(b)
	return string(data)
}

// nowMs is a small helper kept here to avoid importing time in servers package.
func NowMs() int64 {
	return time.Now().UnixMilli()
}
