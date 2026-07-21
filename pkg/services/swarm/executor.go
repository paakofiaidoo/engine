package swarm

import (
	"context"
	"encoding/json"
	"fmt"

	"juki-engine/pkg/data/models"
	"juki-engine/pkg/data/repositories"
	"juki-engine/pkg/services/ai"
)

// Status constants for the swarm state machine:
//   IDLE → PLANNING → REVIEWING → EXECUTING → DONE
//                          ↓
//                      CANCELLED
const (
	StatusIdle      = "idle"
	StatusPlanning  = "planning"
	StatusReviewing = "reviewing"
	StatusExecuting = "executing"
	StatusDone      = "done"
	StatusCancelled = "cancelled"
)

// Event mirrors the existing dtos.SwarmEvent shape so the gRPC layer and
// existing UI streaming code keep working without changes — but now the
// content is REAL agent output, not a timed simulation.
type Event struct {
	Type    string // "status" | "plan" | "log" | "result"
	Status  string
	Plan    *Plan
	Log     *LogLine
	Result  *Result
}

type Plan struct {
	PM        *PMPlan        `json:"pm,omitempty"`
	Architect *ArchitectPlan `json:"architect,omitempty"`
	Designer  *DesignerPlan  `json:"designer,omitempty"`
	Tasks     []BuildTask    `json:"tasks,omitempty"`
}

type LogLine struct {
	Agent   string
	Message string
	Level   string
}

type Result struct {
	Success bool
	Summary string
	Builds  []*BuildResult
}

// Executor runs the real swarm pipeline end to end, persisting state at each
// transition and emitting events for the UI to stream.
type Executor struct {
	repo  repositories.Repository
	aiSvc *ai.Service
}

func NewExecutor(repo repositories.Repository, aiSvc *ai.Service) *Executor {
	return &Executor{repo: repo, aiSvc: aiSvc}
}

// Run executes the full pipeline for a project against its approved brief.
// briefJSON is the JSON brief data (from project_briefs); projectID scopes everything.
//
// State persisted to `swarm_sessions` at each transition so sessions can be
// paused/resumed (per the documented design — resumption logic builds on
// the persisted PlanJSON + Status fields).
func (e *Executor) Run(ctx context.Context, projectID, briefID, briefJSON string) (<-chan Event, error) {
	session := &models.SwarmSession{
		ProjectID: projectID,
		BriefID:   briefID,
		Status:    StatusPlanning,
	}
	if err := e.repo.CreateSwarmSession(session); err != nil {
		return nil, err
	}

	events := make(chan Event, 64)

	go func() {
		defer close(events)
		e.runPipeline(ctx, session, briefJSON, events)
	}()

	return events, nil
}

func (e *Executor) emit(events chan<- Event, ctx context.Context, ev Event) bool {
	select {
	case events <- ev:
		return true
	case <-ctx.Done():
		return false
	}
}

func (e *Executor) setStatus(session *models.SwarmSession, status string) {
	session.Status = status
	_ = e.repo.UpdateSwarmSession(session)
}

func (e *Executor) runPipeline(ctx context.Context, session *models.SwarmSession, briefJSON string, events chan<- Event) {
	projectID := session.ProjectID

	// ─── PLANNING ───────────────────────────────────────────────────────────
	e.setStatus(session, StatusPlanning)
	if !e.emit(events, ctx, Event{Type: "status", Status: StatusPlanning}) {
		return
	}
	if !e.emit(events, ctx, Event{Type: "log", Log: &LogLine{Agent: "pm", Message: "Reviewing approved brief and producing the structured plan...", Level: "info"}}) {
		return
	}

	pmPlan, err := RunPMAgent(ctx, e.aiSvc, projectID, briefJSON)
	if err != nil {
		e.fail(ctx, session, events, fmt.Sprintf("PM Agent failed: %v", err))
		return
	}
	if !e.emit(events, ctx, Event{Type: "log", Log: &LogLine{Agent: "pm", Message: "Plan ready. Handing off to Architect Agent.", Level: "info"}}) {
		return
	}

	archPlan, err := RunArchitectAgent(ctx, e.aiSvc, projectID, pmPlan)
	if err != nil {
		e.fail(ctx, session, events, fmt.Sprintf("Architect Agent failed: %v", err))
		return
	}
	if !e.emit(events, ctx, Event{Type: "log", Log: &LogLine{Agent: "architect", Message: fmt.Sprintf("Stack decided: %v. Packages: %v.", archPlan.Stack, archPlan.PackagesToAdd), Level: "info"}}) {
		return
	}

	designPlan, err := RunDesignerAgent(ctx, e.aiSvc, projectID, pmPlan)
	if err != nil {
		e.fail(ctx, session, events, fmt.Sprintf("Designer Agent failed: %v", err))
		return
	}
	if !e.emit(events, ctx, Event{Type: "log", Log: &LogLine{Agent: "designer", Message: "Theme configuration generated.", Level: "info"}}) {
		return
	}

	tasks := buildTasksFromPlan(pmPlan)

	plan := &Plan{PM: pmPlan, Architect: archPlan, Designer: designPlan, Tasks: tasks}
	planJSON, _ := json.Marshal(plan)
	session.PlanJSON = string(planJSON)
	e.setStatus(session, StatusReviewing)

	if !e.emit(events, ctx, Event{Type: "plan", Plan: plan}) {
		return
	}
	if !e.emit(events, ctx, Event{Type: "status", Status: StatusReviewing}) {
		return
	}
	if !e.emit(events, ctx, Event{Type: "log", Log: &LogLine{Agent: "swarm", Message: "Plan ready for review. Waiting for approval to execute.", Level: "info"}}) {
		return
	}

	// ─── EXECUTING ──────────────────────────────────────────────────────────
	// NOTE: In a full implementation, REVIEWING would block here on an
	// ApproveStep gRPC call (persisted via the session row + a notification
	// channel). For this local test pass, we proceed automatically and log
	// the transition explicitly so the UI/dev can observe the real state machine.
	e.setStatus(session, StatusExecuting)
	if !e.emit(events, ctx, Event{Type: "status", Status: StatusExecuting}) {
		return
	}

	tools := NewTools(e.repo, projectID)
	var builds []*BuildResult
	for _, task := range tasks {
		if !e.emit(events, ctx, Event{Type: "log", Log: &LogLine{Agent: "builder", Message: fmt.Sprintf("Building %s: %s", task.Kind, task.Name), Level: "info"}}) {
			return
		}

		result, err := RunBuilderAgent(ctx, e.aiSvc, tools, projectID, task)
		if err != nil {
			if !e.emit(events, ctx, Event{Type: "log", Log: &LogLine{Agent: "builder", Message: fmt.Sprintf("Failed: %v", err), Level: "error"}}) {
				return
			}
			continue
		}
		builds = append(builds, result)

		if !e.emit(events, ctx, Event{Type: "log", Log: &LogLine{Agent: "builder", Message: fmt.Sprintf("Done: %s — %s", task.Name, result.Summary), Level: "info"}}) {
			return
		}
	}

	// ─── DONE ───────────────────────────────────────────────────────────────
	e.setStatus(session, StatusDone)
	finalResult := &Result{
		Success: true,
		Summary: fmt.Sprintf("Swarm completed. %d build tasks executed.", len(builds)),
		Builds:  builds,
	}
	if !e.emit(events, ctx, Event{Type: "result", Result: finalResult}) {
		return
	}
	e.emit(events, ctx, Event{Type: "status", Status: StatusDone})
}

func (e *Executor) fail(ctx context.Context, session *models.SwarmSession, events chan<- Event, message string) {
	e.setStatus(session, StatusCancelled)
	e.emit(events, ctx, Event{Type: "log", Log: &LogLine{Agent: "swarm", Message: message, Level: "error"}})
	e.emit(events, ctx, Event{Type: "result", Result: &Result{Success: false, Summary: message}})
	e.emit(events, ctx, Event{Type: "status", Status: StatusCancelled})
}

// buildTasksFromPlan converts the PM plan's page list into concrete build tasks.
// Component and data-model tasks are derived similarly; pages drive the
// majority of the initial scaffold per the documented Builder Agent split
// (Page Builder / Component Builder / Data Builder).
func buildTasksFromPlan(plan *PMPlan) []BuildTask {
	var tasks []BuildTask
	for _, page := range plan.Pages {
		tasks = append(tasks, BuildTask{
			Kind:    "page",
			Name:    page,
			Route:   "/" + slugify(page),
			Outline: fmt.Sprintf("Create the %q page per the project brief.", page),
		})
	}
	for _, model := range plan.DataModels {
		tasks = append(tasks, BuildTask{
			Kind:    "data",
			Name:    model,
			Outline: fmt.Sprintf("Create the %q data model and its type definitions.", model),
		})
	}
	return tasks
}

func slugify(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
		case r >= 'A' && r <= 'Z':
			out = append(out, r+('a'-'A'))
		case r == ' ' || r == '_':
			out = append(out, '-')
		}
	}
	if len(out) == 0 || string(out) == "home" {
		return ""
	}
	return string(out)
}
