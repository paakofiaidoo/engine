package services

import (
	"context"
	"fmt"

	"juki-engine/pkg/data/dtos"
	"juki-engine/pkg/services/swarm"
)

// RunSwarm runs the real multi-agent pipeline (PM → Architect → Designer → Builder)
// against the project's most recently approved brief, converting the internal
// swarm.Event stream into the existing dtos.SwarmEvent shape so the gRPC layer
// (pkg/servers/engine.go RunSwarm handler) and UI streaming code keep working
// without any changes.
//
// "status" events (idle/planning/reviewing/executing/done/cancelled) are folded
// into SwarmEventLog entries (StepID: "status") since the existing DTO has no
// dedicated status event type — the UI already renders log lines.
func (s *service) RunSwarm(projectID, prompt string) (<-chan dtos.SwarmEvent, error) {
	brief, briefErr := s.repository.GetLatestBrief(projectID)
	if briefErr != nil || brief == nil {
		return nil, fmt.Errorf("no approved project brief found for project %s — run Sprint Planning first", projectID)
	}

	executor := swarm.NewExecutor(s.repository, s.aiService)

	internalEvents, err := executor.Run(context.Background(), projectID, brief.ID, brief.BriefJSON)
	if err != nil {
		return nil, err
	}

	events := make(chan dtos.SwarmEvent, 64)

	go func() {
		defer close(events)

		for ev := range internalEvents {
			switch ev.Type {
			case "status":
				events <- dtos.SwarmEvent{
					Type: dtos.SwarmEventLog,
					Log:  &dtos.SwarmLog{StepID: "status", Message: fmt.Sprintf("Swarm status: %s", ev.Status), Level: "info"},
				}

			case "plan":
				events <- dtos.SwarmEvent{Type: dtos.SwarmEventPlan, Plan: convertPlan(ev.Plan)}

			case "log":
				if ev.Log != nil {
					events <- dtos.SwarmEvent{
						Type: dtos.SwarmEventLog,
						Log:  &dtos.SwarmLog{StepID: ev.Log.Agent, Message: ev.Log.Message, Level: ev.Log.Level},
					}
				}

			case "result":
				if ev.Result != nil {
					events <- dtos.SwarmEvent{
						Type: dtos.SwarmEventResult,
						Result: &dtos.SwarmResult{
							Success: ev.Result.Success,
							Message: ev.Result.Summary,
						},
					}
				}
			}
		}
	}()

	return events, nil
}

// convertPlan maps the real swarm Plan (PM/Architect/Designer/Tasks) into the
// existing dtos.SwarmPlan{Steps} shape expected by the gRPC layer and UI.
func convertPlan(plan *swarm.Plan) *dtos.SwarmPlan {
	if plan == nil {
		return &dtos.SwarmPlan{}
	}

	var steps []dtos.SwarmStep
	for i, task := range plan.Tasks {
		steps = append(steps, dtos.SwarmStep{
			ID:          fmt.Sprintf("task-%d", i+1),
			Description: fmt.Sprintf("[%s] %s — %s", task.Kind, task.Name, task.Outline),
			Status:      "pending",
		})
	}

	return &dtos.SwarmPlan{Steps: steps}
}
