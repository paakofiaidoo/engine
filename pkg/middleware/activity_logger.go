package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"juki-engine/pkg/data/models"
	"juki-engine/pkg/data/repositories"

	"connectrpc.com/connect"
)

// mutationWhitelist contains the gRPC methods we want to log.
// High-frequency "Get" or "Watch" methods are excluded.
var mutationWhitelist = map[string]bool{
	// Project
	"/juki.engine.v1.EngineService/CreateProject": true,
	"/juki.engine.v1.EngineService/UpdateProject": true,
	"/juki.engine.v1.EngineService/DeleteProject": true,

	// Page/Layout
	"/juki.engine.v1.EngineService/CreatePage": true,
	"/juki.engine.v1.EngineService/UpdatePage": true,

	// Components
	"/juki.engine.v1.EngineService/CreateComponent": true,
	// Add more as needed
}

func NewActivityLogger(repo repositories.Repository) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			procedure := req.Spec().Procedure

			// 1. Check Whitelist (Filtering)
			// Only log if it's in our mutation list
			// OR if it returns an ERROR (we might want to log all errors regardless)
			isMutation := mutationWhitelist[procedure]

			resp, err := next(ctx, req)
			duration := time.Since(start)

			// 2. Decide to Log
			if isMutation || err != nil {
				go func() {
					logEntry := &models.ActivityLog{
						ActionType: getActionType(procedure, err),
						Method:     procedure,
						StatusCode: 200, // Default OK
						DurationMs: duration.Milliseconds(),
						Timestamp:  start,
					}

					// Serialize Payload
					if b, err := json.Marshal(req.Any()); err == nil {
						logEntry.Payload = string(b)
					}

					// Serialize Response or Error
					if err != nil {
						logEntry.ActionType = "ERROR"
						logEntry.StatusCode = 500 // Generic error code
						logEntry.Response = err.Error()
					} else if b, err := json.Marshal(resp.Any()); err == nil {
						logEntry.Response = string(b)
					}

					// Extract ProjectID from request if available (generic inspection)
					// This is tricky with `Any`, we rely on it being present in the JSON payload
					// and potentially extracting it. For MVP, we skip strictly parsing ProjectID from generic struct
					// unless we cast to specific types, which is hard in middleware.
					// We could unmarshal to a map to find "project_id".

					_ = repo.CreateActivityLog(logEntry)
				}()
			}

			if isMutation {
				fmt.Printf("[Activity] %s took %v\n", procedure, duration)
			}

			return resp, err
		}
	}
}

func getActionType(method string, err error) string {
	if err != nil {
		return "ERROR"
	}
	if strings.Contains(method, "Create") {
		return "CREATE"
	}
	if strings.Contains(method, "Update") {
		return "UPDATE"
	}
	if strings.Contains(method, "Delete") {
		return "DELETE"
	}
	return "MUTATION"
}
