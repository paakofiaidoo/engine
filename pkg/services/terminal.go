package services

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"juki-engine/pkg/data/enums"
	"juki-engine/pkg/data/models"
	"juki-engine/pkg/data/repositories"
	enginev1 "juki-engine/pkg/gen/juki/engine/v1"

	"github.com/creack/pty"
)

type TerminalService interface {
	StartTerminal(projectID string, cmdType enginev1.CommandType, customArgs string, outChan chan<- []byte) error
	SendCommand(projectID string, cmd string) error
	ResizeTerminal(projectID string, rows, cols int) error
	KillTerminal(projectID string) error
}

type terminalSession struct {
	cmd     *exec.Cmd
	ptmx    *os.File
	outChan chan<- []byte
}

type terminalService struct {
	repo     repositories.Repository
	sessions map[string]*terminalSession
	mu       sync.Mutex
}

func NewTerminalService(repo repositories.Repository) TerminalService {
	return &terminalService{
		repo:     repo,
		sessions: make(map[string]*terminalSession),
	}
}

func (s *terminalService) StartTerminal(projectID string, cmdType enginev1.CommandType, customArgs string, outChan chan<- []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Check existing session
	if _, exists := s.sessions[projectID]; exists {
		s.killTerminalLocked(projectID)
	}

	project, err := s.repo.GetProject(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// 2. Resolve Command from Enum
	// cmdKey := cmdType.String()
	// The generated enum string mimics COMMAND_TYPE_DEV, so we strip prefix or map manually.
	// We'll trust our manual mapping for now as it's cleaner.

	var commandTemplate string
	var isSafe bool

	switch cmdType {
	case enginev1.CommandType_COMMAND_TYPE_DEV:
		commandTemplate = enums.CommandRegistry["DEV"].Template
		isSafe = enums.CommandRegistry["DEV"].Safe
	case enginev1.CommandType_COMMAND_TYPE_BUILD:
		commandTemplate = enums.CommandRegistry["BUILD"].Template
		isSafe = enums.CommandRegistry["BUILD"].Safe
	case enginev1.CommandType_COMMAND_TYPE_INSTALL:
		commandTemplate = enums.CommandRegistry["INSTALL"].Template
		// Validate customArgs for install
		if !enums.IsSafeArg(customArgs) {
			return fmt.Errorf("unsafe arguments detected for install command")
		}
		commandTemplate = fmt.Sprintf(commandTemplate, customArgs)
		isSafe = true // Now deemed safe after check
	case enginev1.CommandType_COMMAND_TYPE_GIT_PUSH:
		commandTemplate = enums.CommandRegistry["GIT_PUSH"].Template
		isSafe = enums.CommandRegistry["GIT_PUSH"].Safe
	default:
		return fmt.Errorf("unsupported command type: %v", cmdType)
	}

	if !isSafe {
		return fmt.Errorf("command type %v is marked as unsafe", cmdType)
	}

	// 3. Launch PTY
	// commandTemplate e.g. "npm run dev". Split by space for exec.Command?
	// exec.Command("bash", "-c", commandTemplate) is flexible but requires bash.
	c := exec.Command("bash", "-c", commandTemplate)
	c.Dir = project.Path

	ptmx, err := pty.Start(c)
	if err != nil {
		return err
	}

	// 4. Log to DB
	activity := &models.TerminalActivity{
		ProjectID: projectID,
		Status:    "RUNNING",
		Type:      cmdType.String(),
		PID:       c.Process.Pid,
		StartedAt: time.Now(),
	}
	_ = s.repo.CreateTerminalActivity(activity)

	session := &terminalSession{
		cmd:     c,
		ptmx:    ptmx,
		outChan: outChan,
	}
	s.sessions[projectID] = session

	// 5. Read Loop
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
				fmt.Printf("Error reading pty: %v\n", err)
				break
			}
			data := make([]byte, n)
			copy(data, buf[:n])

			select {
			case outChan <- data:
			default:
			}
		}

		// Update DB on exit
		now := time.Now()
		activity.StoppedAt = &now
		activity.Status = "STOPPED"
		_ = s.repo.UpdateTerminalActivity(activity)

		s.KillTerminal(projectID)
	}()

	return nil
}

func (s *terminalService) SendCommand(projectID string, cmd string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[projectID]
	if !ok {
		return fmt.Errorf("no terminal for project %s", projectID)
	}

	_, err := session.ptmx.Write([]byte(cmd))
	return err
}

func (s *terminalService) ResizeTerminal(projectID string, rows, cols int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[projectID]
	if !ok {
		return fmt.Errorf("no terminal for project %s", projectID)
	}

	return pty.Setsize(session.ptmx, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
}

func (s *terminalService) KillTerminal(projectID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.killTerminalLocked(projectID)
}

func (s *terminalService) killTerminalLocked(projectID string) error {
	session, ok := s.sessions[projectID]
	if !ok {
		return nil
	}

	delete(s.sessions, projectID)

	// Close ptmx first
	_ = session.ptmx.Close()

	if session.cmd != nil && session.cmd.Process != nil {
		_ = session.cmd.Process.Kill()
	}

	close(session.outChan)
	return nil
}
