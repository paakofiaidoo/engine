package api

import (
	"context"
	"encoding/json"

	enginev1 "juki-engine/pkg/gen/juki/engine/v1"
	"juki-engine/pkg/services/sprint"

	"connectrpc.com/connect"
)

// SprintServer implements SprintPlanningServiceHandler.
type SprintServer struct {
	svc *sprint.Service
}

func NewSprintServer(svc *sprint.Service) *SprintServer {
	return &SprintServer{svc: svc}
}

func briefToProto(b *sprint.BriefData, id, projectID string, version int32, createdAtMs int64) *enginev1.ProjectBriefData {
	if b == nil {
		return nil
	}
	return &enginev1.ProjectBriefData{
		Id:                id,
		ProjectId:         projectID,
		Version:           version,
		Description:       b.Description,
		Users:             b.Users,
		Pages:             b.Pages,
		Plugins:           b.Plugins,
		DataModels:        b.DataModels,
		ThemeStyle:        b.Theme.Style,
		ThemePrimaryColor: b.Theme.PrimaryColor,
		CreatedAtMs:       createdAtMs,
	}
}

func (s *SprintServer) StartSession(
	ctx context.Context,
	req *connect.Request[enginev1.StartSessionRequest],
) (*connect.Response[enginev1.StartSessionResponse], error) {
	conversationID, opening, err := s.svc.StartSession(ctx, req.Msg.ProjectId, req.Msg.Description)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&enginev1.StartSessionResponse{
		ConversationId: conversationID,
		OpeningMessage: opening,
	}), nil
}

func (s *SprintServer) SendPlanningMessage(
	ctx context.Context,
	req *connect.Request[enginev1.SendPlanningMessageRequest],
	stream *connect.ServerStream[enginev1.SendPlanningMessageResponse],
) error {
	reply, brief, err := s.svc.SendMessage(ctx, req.Msg.ProjectId, req.Msg.ConversationId, req.Msg.Message)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	// Stream the reply back as a single chunk (the underlying ai.Service call
	// is currently non-streamed for sprint planning — keeps brief extraction
	// simple and reliable). A future pass can wire true token streaming here.
	if err := stream.Send(&enginev1.SendPlanningMessageResponse{Delta: reply}); err != nil {
		return err
	}

	var draftBrief *enginev1.ProjectBriefData
	if brief != nil {
		draftBrief = briefToProto(brief, "", req.Msg.ProjectId, 0, sprint.NowMs())
	}

	return stream.Send(&enginev1.SendPlanningMessageResponse{
		Done:       true,
		DraftBrief: draftBrief,
	})
}

func (s *SprintServer) ApproveBrief(
	ctx context.Context,
	req *connect.Request[enginev1.ApproveBriefRequest],
) (*connect.Response[enginev1.ApproveBriefResponse], error) {
	saved, err := s.svc.ApproveBrief(req.Msg.ProjectId, req.Msg.BriefJson)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var parsed sprint.BriefData
	_ = json.Unmarshal([]byte(saved.BriefJSON), &parsed)

	return connect.NewResponse(&enginev1.ApproveBriefResponse{
		Brief: briefToProto(&parsed, saved.ID, saved.ProjectID, int32(saved.Version), saved.CreatedAt.UnixMilli()),
	}), nil
}

func (s *SprintServer) GetBrief(
	ctx context.Context,
	req *connect.Request[enginev1.GetBriefRequest],
) (*connect.Response[enginev1.GetBriefResponse], error) {
	brief, found := s.svc.GetBrief(req.Msg.ProjectId)
	if !found {
		return connect.NewResponse(&enginev1.GetBriefResponse{Found: false}), nil
	}

	var parsed sprint.BriefData
	_ = json.Unmarshal([]byte(brief.BriefJSON), &parsed)

	return connect.NewResponse(&enginev1.GetBriefResponse{
		Found: true,
		Brief: briefToProto(&parsed, brief.ID, brief.ProjectID, int32(brief.Version), brief.CreatedAt.UnixMilli()),
	}), nil
}
