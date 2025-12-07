package services

import (
	"fmt"
	"juki-engine/pkg/data/dtos"
	"time"
)

func (s *service) RunSwarm(projectID, prompt string) (<-chan dtos.SwarmEvent, error) {
	events := make(chan dtos.SwarmEvent)

	go func() {
		defer close(events)

		// 1. Simulate Planning
		time.Sleep(1 * time.Second)
		plan := &dtos.SwarmPlan{
			Steps: []dtos.SwarmStep{
				{ID: "step-1", Description: "Analyze Requirements", Status: "pending"},
				{ID: "step-2", Description: "Generate Component", Status: "pending"},
				{ID: "step-3", Description: "Update Project", Status: "pending"},
			},
		}
		events <- dtos.SwarmEvent{Type: dtos.SwarmEventPlan, Plan: plan}

		// 2. Simulate Execution
		steps := []string{"step-1", "step-2", "step-3"}
		for _, stepID := range steps {
			// Start Step
			events <- dtos.SwarmEvent{
				Type: dtos.SwarmEventLog,
				Log:  &dtos.SwarmLog{StepID: stepID, Message: fmt.Sprintf("Starting %s...", stepID), Level: "info"},
			}
			time.Sleep(1 * time.Second)

			// Log progress
			events <- dtos.SwarmEvent{
				Type: dtos.SwarmEventLog,
				Log:  &dtos.SwarmLog{StepID: stepID, Message: "Working on it...", Level: "info"},
			}
			time.Sleep(1 * time.Second)

			// Complete Step
			events <- dtos.SwarmEvent{
				Type: dtos.SwarmEventLog,
				Log:  &dtos.SwarmLog{StepID: stepID, Message: "Done.", Level: "info"},
			}
		}

		// 3. Result
		events <- dtos.SwarmEvent{
			Type: dtos.SwarmEventResult,
			Result: &dtos.SwarmResult{
				Success: true,
				Message: "Swarm execution completed successfully.",
			},
		}
	}()

	return events, nil
}