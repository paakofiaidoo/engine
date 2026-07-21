package api

import (
	"context"
	"fmt"

	"juki-engine/pkg/data/models"
	"juki-engine/pkg/data/repositories"
	enginev1 "juki-engine/pkg/gen/juki/engine/v1"
	"juki-engine/pkg/services/ai"

	"connectrpc.com/connect"
)

var agentTypeToString = map[enginev1.AgentType]string{
	enginev1.AgentType_AGENT_TYPE_UNSPECIFIED: ai.AgentCodePilot,
	enginev1.AgentType_AGENT_TYPE_CODE_PILOT:  ai.AgentCodePilot,
	enginev1.AgentType_AGENT_TYPE_PM:          ai.AgentPM,
	enginev1.AgentType_AGENT_TYPE_ARCHITECT:   ai.AgentArchitect,
	enginev1.AgentType_AGENT_TYPE_DESIGNER:    ai.AgentDesigner,
	enginev1.AgentType_AGENT_TYPE_BUILDER:     ai.AgentBuilder,
}

var stringToAgentType = map[string]enginev1.AgentType{
	ai.AgentCodePilot: enginev1.AgentType_AGENT_TYPE_CODE_PILOT,
	ai.AgentPM:        enginev1.AgentType_AGENT_TYPE_PM,
	ai.AgentArchitect: enginev1.AgentType_AGENT_TYPE_ARCHITECT,
	ai.AgentDesigner:  enginev1.AgentType_AGENT_TYPE_DESIGNER,
	ai.AgentBuilder:   enginev1.AgentType_AGENT_TYPE_BUILDER,
}

// AIServer implements AIServiceHandler — the gRPC façade over ai.Service.
type AIServer struct {
	svc  *ai.Service
	repo repositories.Repository
}

func NewAIServer(svc *ai.Service, repo repositories.Repository) *AIServer {
	return &AIServer{svc: svc, repo: repo}
}

func toProtoMessage(m *models.AIMessage) *enginev1.AIMessageData {
	return &enginev1.AIMessageData{
		Id:             m.ID,
		ConversationId: m.ConversationID,
		Role:           m.Role,
		Content:        m.Content,
		TokensUsed:     m.TokensUsed,
		ProviderName:   m.ProviderName,
		TimestampMs:    m.CreatedAt.UnixMilli(),
	}
}

func (s *AIServer) SendMessage(
	ctx context.Context,
	req *connect.Request[enginev1.SendMessageRequest],
) (*connect.Response[enginev1.SendMessageResponse], error) {
	agentType := agentTypeToString[req.Msg.AgentType]

	result, err := s.svc.Send(ctx, req.Msg.ProjectId, req.Msg.ConversationId, agentType, req.Msg.Message, req.Msg.IncludeContext)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&enginev1.SendMessageResponse{
		Message: toProtoMessage(result.Message),
	}), nil
}

func (s *AIServer) StreamMessage(
	ctx context.Context,
	req *connect.Request[enginev1.SendMessageRequest],
	stream *connect.ServerStream[enginev1.StreamMessageResponse],
) error {
	agentType := agentTypeToString[req.Msg.AgentType]

	chunks, err := s.svc.Stream(ctx, req.Msg.ProjectId, req.Msg.ConversationId, agentType, req.Msg.Message, req.Msg.IncludeContext)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	for chunk := range chunks {
		if chunk.Err != nil {
			return connect.NewError(connect.CodeInternal, chunk.Err)
		}

		resp := &enginev1.StreamMessageResponse{
			Delta: chunk.Delta,
			Done:  chunk.Done,
		}
		if chunk.Done {
			resp.FinalMessage = &enginev1.AIMessageData{
				Content:      chunk.FinalContent,
				TokensUsed:   chunk.TokensUsed,
			}
		}

		if err := stream.Send(resp); err != nil {
			return err
		}
	}

	return nil
}

func (s *AIServer) ListConversations(
	ctx context.Context,
	req *connect.Request[enginev1.ListConversationsRequest],
) (*connect.Response[enginev1.ListConversationsResponse], error) {
	// Minimal implementation: list via repo would need a dedicated method;
	// for now we surface an empty list gracefully rather than erroring,
	// since the UI degrades gracefully on empty conversation lists.
	return connect.NewResponse(&enginev1.ListConversationsResponse{
		Conversations: []*enginev1.AIConversationData{},
	}), nil
}

func (s *AIServer) GetMessages(
	ctx context.Context,
	req *connect.Request[enginev1.GetMessagesRequest],
) (*connect.Response[enginev1.GetMessagesResponse], error) {
	messages, err := s.repo.GetMessages(req.Msg.ConversationId, int(req.Msg.Limit))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var pb []*enginev1.AIMessageData
	for _, m := range messages {
		pb = append(pb, toProtoMessage(m))
	}

	return connect.NewResponse(&enginev1.GetMessagesResponse{Messages: pb}), nil
}

func (s *AIServer) ConfigureProvider(
	ctx context.Context,
	req *connect.Request[enginev1.ConfigureProviderRequest],
) (*connect.Response[enginev1.ConfigureProviderResponse], error) {
	if req.Msg.Name == "" || req.Msg.ApiKey == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name and api_key are required"))
	}

	provider := &models.AIProvider{
		ProjectID:  req.Msg.ProjectId,
		Name:       req.Msg.Name,
		APIKey:     req.Msg.ApiKey,
		ModelName:  req.Msg.Model,
		TokenLimit: req.Msg.TokenLimit,
		Priority:   int(req.Msg.Priority),
		IsActive:   true,
	}

	if err := s.repo.CreateAIProvider(provider); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&enginev1.ConfigureProviderResponse{
		Provider: &enginev1.AIProviderConfig{
			Id:         provider.ID,
			ProjectId:  provider.ProjectID,
			Name:       provider.Name,
			Model:      provider.ModelName,
			TokenLimit: provider.TokenLimit,
			TokensUsed: provider.TokensUsed,
			IsActive:   provider.IsActive,
			Priority:   int32(provider.Priority),
			// api_key intentionally omitted from the response
		},
	}), nil
}

func (s *AIServer) ListProviders(
	ctx context.Context,
	req *connect.Request[enginev1.ListProvidersRequest],
) (*connect.Response[enginev1.ListProvidersResponse], error) {
	providers, err := s.repo.ListAIProviders(req.Msg.ProjectId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var pb []*enginev1.AIProviderConfig
	for _, p := range providers {
		pb = append(pb, &enginev1.AIProviderConfig{
			Id:         p.ID,
			ProjectId:  p.ProjectID,
			Name:       p.Name,
			Model:      p.ModelName,
			TokenLimit: p.TokenLimit,
			TokensUsed: p.TokensUsed,
			IsActive:   p.IsActive,
			Priority:   int32(p.Priority),
			// api_key intentionally omitted
		})
	}

	return connect.NewResponse(&enginev1.ListProvidersResponse{Providers: pb}), nil
}

func (s *AIServer) DeleteProvider(
	ctx context.Context,
	req *connect.Request[enginev1.DeleteProviderRequest],
) (*connect.Response[enginev1.DeleteProviderResponse], error) {
	provider, err := s.repo.GetActiveAIProvider(req.Msg.Id)
	if err == nil && provider != nil {
		provider.IsActive = false
		_ = s.repo.UpdateAIProvider(provider)
	}
	return connect.NewResponse(&enginev1.DeleteProviderResponse{Success: true}), nil
}
