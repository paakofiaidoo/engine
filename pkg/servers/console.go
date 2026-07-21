package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"juki-engine/pkg/data/models"
	"juki-engine/pkg/data/repositories"
	enginev1 "juki-engine/pkg/gen/juki/engine/v1"

	"connectrpc.com/connect"
)

// levelToString / stringToLevel map between proto enum and DB string representation.
var levelToString = map[enginev1.LogLevel]string{
	enginev1.LogLevel_LOG_LEVEL_UNSPECIFIED: "log",
	enginev1.LogLevel_LOG_LEVEL_LOG:         "log",
	enginev1.LogLevel_LOG_LEVEL_INFO:        "info",
	enginev1.LogLevel_LOG_LEVEL_WARN:        "warn",
	enginev1.LogLevel_LOG_LEVEL_ERROR:       "error",
	enginev1.LogLevel_LOG_LEVEL_DEBUG:       "debug",
}

var stringToLevel = map[string]enginev1.LogLevel{
	"log":   enginev1.LogLevel_LOG_LEVEL_LOG,
	"info":  enginev1.LogLevel_LOG_LEVEL_INFO,
	"warn":  enginev1.LogLevel_LOG_LEVEL_WARN,
	"error": enginev1.LogLevel_LOG_LEVEL_ERROR,
	"debug": enginev1.LogLevel_LOG_LEVEL_DEBUG,
}

// ConsoleServer implements ConsoleServiceHandler.
// It persists logs to the DB and fans them out to live StreamLogs subscribers.
type ConsoleServer struct {
	repo repositories.Repository

	mu          sync.RWMutex
	subscribers map[string][]chan *models.ConsoleLog // keyed by projectID
}

func NewConsoleServer(repo repositories.Repository) *ConsoleServer {
	return &ConsoleServer{
		repo:        repo,
		subscribers: make(map[string][]chan *models.ConsoleLog),
	}
}

func toProtoEntry(e *models.ConsoleLog) *enginev1.LogEntry {
	level, ok := stringToLevel[e.Level]
	if !ok {
		level = enginev1.LogLevel_LOG_LEVEL_LOG
	}
	return &enginev1.LogEntry{
		Id:             e.ID,
		ProjectId:      e.ProjectID,
		SessionId:      e.SessionID,
		Level:          level,
		Message:        e.Message,
		ArgsJson:       e.ArgsJSON,
		TimestampMs:    e.TimestampMs,
		SourceLocation: e.SourceLocation,
	}
}

func (s *ConsoleServer) PushLog(
	ctx context.Context,
	req *connect.Request[enginev1.PushLogRequest],
) (*connect.Response[enginev1.PushLogResponse], error) {
	entry := req.Msg.Entry
	if entry == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("entry is required"))
	}

	ts := entry.TimestampMs
	if ts == 0 {
		ts = time.Now().UnixMilli()
	}

	model := &models.ConsoleLog{
		ProjectID:      entry.ProjectId,
		SessionID:      entry.SessionId,
		Level:          levelToString[entry.Level],
		Message:        entry.Message,
		ArgsJSON:       entry.ArgsJson,
		TimestampMs:    ts,
		SourceLocation: entry.SourceLocation,
	}

	if err := s.repo.CreateConsoleLog(model); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Fan out to live subscribers (non-blocking)
	s.broadcast(model)

	return connect.NewResponse(&enginev1.PushLogResponse{Id: model.ID}), nil
}

func (s *ConsoleServer) StreamLogs(
	ctx context.Context,
	req *connect.Request[enginev1.StreamLogsRequest],
	stream *connect.ServerStream[enginev1.StreamLogsResponse],
) error {
	projectID := req.Msg.ProjectId
	ch := make(chan *models.ConsoleLog, 100)

	s.subscribe(projectID, ch)
	defer s.unsubscribe(projectID, ch)

	for {
		select {
		case <-ctx.Done():
			return nil
		case entry, ok := <-ch:
			if !ok {
				return nil
			}
			if req.Msg.SessionId != "" && entry.SessionID != req.Msg.SessionId {
				continue
			}
			if err := stream.Send(&enginev1.StreamLogsResponse{Entry: toProtoEntry(entry)}); err != nil {
				return err
			}
		}
	}
}

func (s *ConsoleServer) GetLogs(
	ctx context.Context,
	req *connect.Request[enginev1.GetLogsRequest],
) (*connect.Response[enginev1.GetLogsResponse], error) {
	levelFilter := levelToString[req.Msg.LevelFilter]
	if req.Msg.LevelFilter == enginev1.LogLevel_LOG_LEVEL_UNSPECIFIED {
		levelFilter = ""
	}

	entries, total, err := s.repo.GetConsoleLogs(
		req.Msg.ProjectId,
		req.Msg.SessionId,
		int(req.Msg.Limit),
		int(req.Msg.Offset),
		levelFilter,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var pbEntries []*enginev1.LogEntry
	for _, e := range entries {
		pbEntries = append(pbEntries, toProtoEntry(e))
	}

	return connect.NewResponse(&enginev1.GetLogsResponse{
		Entries: pbEntries,
		Total:   int32(total),
	}), nil
}

func (s *ConsoleServer) ClearLogs(
	ctx context.Context,
	req *connect.Request[enginev1.ClearLogsRequest],
) (*connect.Response[enginev1.ClearLogsResponse], error) {
	count, err := s.repo.ClearConsoleLogs(req.Msg.ProjectId, req.Msg.SessionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&enginev1.ClearLogsResponse{DeletedCount: int32(count)}), nil
}

// ─── Subscriber fan-out (internal) ──────────────────────────────────────────

func (s *ConsoleServer) subscribe(projectID string, ch chan *models.ConsoleLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers[projectID] = append(s.subscribers[projectID], ch)
}

func (s *ConsoleServer) unsubscribe(projectID string, ch chan *models.ConsoleLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	subs := s.subscribers[projectID]
	for i, c := range subs {
		if c == ch {
			s.subscribers[projectID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

func (s *ConsoleServer) broadcast(entry *models.ConsoleLog) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ch := range s.subscribers[entry.ProjectID] {
		select {
		case ch <- entry:
		default:
			// Subscriber too slow — drop to avoid blocking the pusher
		}
	}
}
