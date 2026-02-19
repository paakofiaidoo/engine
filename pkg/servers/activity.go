package api

import (
	"context"

	enginev1 "juki-engine/pkg/gen/juki/engine/v1"

	"connectrpc.com/connect"
)

// TODO: Define ActivityService Interface if we want separation like TerminalService
// For now, simpler implementation directly here or just mocking.

type ActivityServer struct {
	// svc services.ActivityService
}

func NewActivityServer() *ActivityServer {
	return &ActivityServer{}
}

func (s *ActivityServer) GetActivityLog(
	ctx context.Context,
	req *connect.Request[enginev1.GetActivityLogRequest],
) (*connect.Response[enginev1.GetActivityLogResponse], error) {
	// TODO: Implement DB query
	return connect.NewResponse(&enginev1.GetActivityLogResponse{
		Logs:  []*enginev1.ActivityLog{},
		Total: 0,
	}), nil
}

func (s *ActivityServer) StreamActivityLog(
	ctx context.Context,
	req *connect.Request[enginev1.StreamActivityLogRequest],
	stream *connect.ServerStream[enginev1.StreamActivityLogResponse],
) error {
	// TODO: Implement log streaming
	// For now, just send a hello log

	// Dummy infinite loop to keep stream open
	select {
	case <-ctx.Done():
		return nil
	}
}
