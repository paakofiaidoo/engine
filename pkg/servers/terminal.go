package api

import (
	"context"
	"fmt"
	"io"

	enginev1 "juki-engine/pkg/gen/juki/engine/v1"
	"juki-engine/pkg/services"

	"connectrpc.com/connect"
)

type TerminalServer struct {
	svc services.TerminalService
}

func NewTerminalServer(svc services.TerminalService) *TerminalServer {
	return &TerminalServer{svc: svc}
}

func (s *TerminalServer) StartTerminal(
	ctx context.Context,
	req *connect.Request[enginev1.StartTerminalRequest],
	stream *connect.ServerStream[enginev1.StartTerminalResponse],
) error {
	fmt.Printf("[API] StartTerminal: ProjectID=%s Type=%v\n", req.Msg.ProjectId, req.Msg.Type)

	outChan := make(chan []byte, 100) // Buffered channel

	// Pass CommandType and CustomArgs to service
	err := s.svc.StartTerminal(req.Msg.ProjectId, req.Msg.Type, req.Msg.CustomArgs, outChan)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	// Ensure cleanup when client disconnects
	defer s.svc.KillTerminal(req.Msg.ProjectId)

	for {
		select {
		case <-ctx.Done():
			return nil
		case data, ok := <-outChan:
			if !ok {
				return nil
			}
			err := stream.Send(&enginev1.StartTerminalResponse{
				Data: data,
			})
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
		}
	}
}

func (s *TerminalServer) SendCommand(
	ctx context.Context,
	req *connect.Request[enginev1.SendCommandRequest],
) (*connect.Response[enginev1.SendCommandResponse], error) {
	// Use Input instead of Command
	err := s.svc.SendCommand(req.Msg.ProjectId, req.Msg.Input)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&enginev1.SendCommandResponse{
		Success: true,
	}), nil
}

func (s *TerminalServer) ResizeTerminal(
	ctx context.Context,
	req *connect.Request[enginev1.ResizeTerminalRequest],
) (*connect.Response[enginev1.ResizeTerminalResponse], error) {
	err := s.svc.ResizeTerminal(req.Msg.ProjectId, int(req.Msg.Rows), int(req.Msg.Cols))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&enginev1.ResizeTerminalResponse{
		Success: true,
	}), nil
}

func (s *TerminalServer) KillTerminal(
	ctx context.Context,
	req *connect.Request[enginev1.KillTerminalRequest],
) (*connect.Response[enginev1.KillTerminalResponse], error) {
	err := s.svc.KillTerminal(req.Msg.ProjectId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&enginev1.KillTerminalResponse{
		Success: true,
	}), nil
}
