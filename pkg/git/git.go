package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git command failed: %s: %w", string(out), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (s *Service) Clone(url, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("destination already exists: %s", dest)
	}
	_, err := s.Run(".", "clone", url, dest)
	return err
}

func (s *Service) Checkout(dir, branch string, create bool) error {
	args := []string{"checkout"}
	if create {
		args = append(args, "-b")
	}
	args = append(args, branch)
	_, err := s.Run(dir, args...)
	return err
}

func (s *Service) Commit(dir, message string) error {
	_, err := s.Run(dir, "add", ".")
	if err != nil {
		return err
	}
	_, err = s.Run(dir, "commit", "-m", message)
	return err
}

func (s *Service) Push(dir, remote, branch string) error {
	_, err := s.Run(dir, "push", remote, branch)
	return err
}

func (s *Service) CurrentBranch(dir string) (string, error) {
	return s.Run(dir, "rev-parse", "--abbrev-ref", "HEAD")
}

// Session Branching Logic

func (s *Service) StartSession(dir, sessionID string) (string, error) {
	branchName := fmt.Sprintf("juki-session-%s", sessionID)
	// Ensure we are on a clean state or handle stash?
	// For now, assume we branch off the current HEAD
	err := s.Checkout(dir, branchName, true)
	return branchName, err
}

func (s *Service) PublishSession(dir, sessionBranch string) error {
	// Switch to juki-dev (create if not exists)
	// This is a simplified flow. In reality, we need to check if juki-dev exists.

	// 1. Fetch latest
	// s.Run(dir, "fetch")

	// 2. Checkout juki-dev
	err := s.Checkout(dir, "juki-dev", false)
	if err != nil {
		// Try creating it
		err = s.Checkout(dir, "juki-dev", true)
		if err != nil {
			return fmt.Errorf("failed to checkout juki-dev: %w", err)
		}
	}

	// 3. Merge session branch (Squash for cleaner history?)
	// Let's use squash to keep juki-dev clean
	_, err = s.Run(dir, "merge", "--squash", sessionBranch)
	if err != nil {
		return fmt.Errorf("failed to merge session: %w", err)
	}

	// 4. Commit the squash
	_, err = s.Run(dir, "commit", "-m", fmt.Sprintf("Publish session %s", sessionBranch))
	if err != nil {
		return fmt.Errorf("failed to commit merge: %w", err)
	}

	return nil
}
