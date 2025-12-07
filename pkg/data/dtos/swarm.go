package dtos

type SwarmEventType string

const (
	SwarmEventPlan   SwarmEventType = "plan"
	SwarmEventLog    SwarmEventType = "log"
	SwarmEventResult SwarmEventType = "result"
)

type SwarmEvent struct {
	Type   SwarmEventType
	Plan   *SwarmPlan
	Log    *SwarmLog
	Result *SwarmResult
}

type SwarmPlan struct {
	Steps []SwarmStep
}

type SwarmStep struct {
	ID          string
	Description string
	Status      string
}

type SwarmLog struct {
	StepID  string
	Message string
	Level   string
}

type SwarmResult struct {
	Success bool
	Message string
}
