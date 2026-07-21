package swarm

import (
	"context"
	"encoding/json"
	"fmt"

	"juki-engine/pkg/services/ai"
)

// Real agent roles, replacing the prior simulated 3-step swarm.
// Each agent is a thin wrapper that:
//   1. Sends a role-specific prompt + current state to the AI service
//   2. Parses the structured JSON the model is asked to emit
//   3. Returns a typed result the executor / state machine can act on
//
// PM Agent → Architect Agent → Designer Agent → Builder Agents (parallel)

// PMPlan is the structured output of the PM Agent (matches ProjectBrief shape).
type PMPlan struct {
	Description string   `json:"description"`
	Pages       []string `json:"pages"`
	Plugins     []string `json:"plugins"`
	DataModels  []string `json:"dataModels"`
	Theme       struct {
		Style        string `json:"style"`
		PrimaryColor string `json:"primaryColor"`
	} `json:"theme"`
}

// ArchitectPlan is the technical decision output.
type ArchitectPlan struct {
	Stack           []string `json:"stack"`           // e.g. ["next.js", "tailwind", "prisma"]
	PackagesToAdd   []string `json:"packagesToAdd"`   // npm packages
	FolderStructure []string `json:"folderStructure"` // proposed dirs/files
}

// DesignerPlan is the theme/visual output.
type DesignerPlan struct {
	Colors     map[string]string `json:"colors"`
	Fonts      []string          `json:"fonts"`
	Spacing    string            `json:"spacing"`
}

// BuildTask is one unit of work for a Builder Agent.
type BuildTask struct {
	Kind    string `json:"kind"` // "page" | "component" | "data"
	Name    string `json:"name"`
	Route   string `json:"route,omitempty"`
	Outline string `json:"outline"`
}

// BuildResult is what a Builder Agent reports back.
type BuildResult struct {
	Task    BuildTask `json:"task"`
	Success bool      `json:"success"`
	Summary string    `json:"summary"`
	ToolCalls []ToolCallRecord `json:"toolCalls"`
}

// ToolCallRecord logs a tool the agent invoked, for transparency in the UI/log.
type ToolCallRecord struct {
	Tool   string `json:"tool"`
	Args   string `json:"args"`
	Result string `json:"result"`
}

// runAgentJSON sends a structured prompt and parses the JSON the model returns
// (expects a fenced ```json block, same convention as the PM sprint planning flow).
func runAgentJSON[T any](ctx context.Context, aiSvc *ai.Service, projectID, agentType, prompt string, out *T) error {
	result, err := aiSvc.Send(ctx, projectID, "", agentType, prompt, true)
	if err != nil {
		return fmt.Errorf("%s agent call failed: %w", agentType, err)
	}

	jsonStr := extractJSONBlock(result.Message.Content)
	if jsonStr == "" {
		return fmt.Errorf("%s agent did not return a JSON block", agentType)
	}

	if err := json.Unmarshal([]byte(jsonStr), out); err != nil {
		return fmt.Errorf("%s agent returned invalid JSON: %w", agentType, err)
	}

	return nil
}

func extractJSONBlock(text string) string {
	const fence = "```json"
	start := indexOf(text, fence)
	if start == -1 {
		return ""
	}
	rest := text[start+len(fence):]
	end := indexOf(rest, "```")
	if end == -1 {
		return ""
	}
	return trimSpace(rest[:end])
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\n' || s[start] == '\t' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\n' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// ─── Agent Runners ───────────────────────────────────────────────────────────

// RunPMAgent produces the structured plan from the approved brief + conversation.
func RunPMAgent(ctx context.Context, aiSvc *ai.Service, projectID, briefJSON string) (*PMPlan, error) {
	prompt := "Based on the approved project brief below, produce the final structured plan " +
		"as a JSON block.\n\nBrief:\n" + briefJSON + "\n\n" +
		"Respond with a ```json block matching: " +
		`{"description":"","pages":[],"plugins":[],"dataModels":[],"theme":{"style":"","primaryColor":""}}`

	var plan PMPlan
	if err := runAgentJSON(ctx, aiSvc, projectID, ai.AgentPM, prompt, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

// RunArchitectAgent decides technical stack & structure from the PM plan.
func RunArchitectAgent(ctx context.Context, aiSvc *ai.Service, projectID string, plan *PMPlan) (*ArchitectPlan, error) {
	planJSON, _ := json.Marshal(plan)
	prompt := "Given this PM plan, decide the technical stack, npm packages to install, " +
		"and folder structure.\n\nPlan:\n" + string(planJSON) + "\n\n" +
		`Respond with a ` + "```json" + ` block matching: {"stack":[],"packagesToAdd":[],"folderStructure":[]}`

	var arch ArchitectPlan
	if err := runAgentJSON(ctx, aiSvc, projectID, ai.AgentArchitect, prompt, &arch); err != nil {
		return nil, err
	}
	return &arch, nil
}

// RunDesignerAgent produces a theme configuration from the PM plan + brand keywords.
func RunDesignerAgent(ctx context.Context, aiSvc *ai.Service, projectID string, plan *PMPlan) (*DesignerPlan, error) {
	planJSON, _ := json.Marshal(plan)
	prompt := "Given this PM plan (note the theme style/color hints), produce a full theme config.\n\n" +
		"Plan:\n" + string(planJSON) + "\n\n" +
		`Respond with a ` + "```json" + ` block matching: {"colors":{"primary":"","secondary":"","background":""},"fonts":[],"spacing":""}`

	var design DesignerPlan
	if err := runAgentJSON(ctx, aiSvc, projectID, ai.AgentDesigner, prompt, &design); err != nil {
		return nil, err
	}
	return &design, nil
}

// RunBuilderAgent executes one build task (page/component/data), calling tools as needed.
// Tool calls are funneled through `tools` which serializes all file mutations
// through the engine — preventing concurrent agents from corrupting each other's edits.
func RunBuilderAgent(ctx context.Context, aiSvc *ai.Service, tools *Tools, projectID string, task BuildTask) (*BuildResult, error) {
	taskJSON, _ := json.Marshal(task)
	prompt := fmt.Sprintf(
		"You have been assigned this build task:\n%s\n\n"+
			"Available tools: %s\n\n"+
			"Describe the steps you'd take and which tools you'd call (with JSON args). "+
			"Respond with a ```json block matching: "+
			`{"toolCalls":[{"tool":"","args":"{}"}],"summary":""}`,
		string(taskJSON), toolNames(tools),
	)

	type plannedCalls struct {
		ToolCalls []struct {
			Tool string `json:"tool"`
			Args string `json:"args"`
		} `json:"toolCalls"`
		Summary string `json:"summary"`
	}

	var planned plannedCalls
	if err := runAgentJSON(ctx, aiSvc, projectID, ai.AgentBuilder, prompt, &planned); err != nil {
		return &BuildResult{Task: task, Success: false, Summary: err.Error()}, err
	}

	var records []ToolCallRecord
	for _, call := range planned.ToolCalls {
		result, err := tools.Call(ctx, call.Tool, call.Args)
		if err != nil {
			result = fmt.Sprintf(`{"error":%q}`, err.Error())
		}
		records = append(records, ToolCallRecord{Tool: call.Tool, Args: call.Args, Result: result})
	}

	return &BuildResult{
		Task:      task,
		Success:   true,
		Summary:   planned.Summary,
		ToolCalls: records,
	}, nil
}

func toolNames(tools *Tools) string {
	var names []string
	for _, spec := range tools.Specs() {
		names = append(names, spec.Name)
	}
	data, _ := json.Marshal(names)
	return string(data)
}
