package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"juki-engine/pkg/git"
)

func main() {
	// 1. Setup Temp Dir
	tempDir, err := os.MkdirTemp("", "juki-git-test")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // Cleanup

	fmt.Printf("Testing Git in: %s\n", tempDir)

	svc := git.NewService()

	// 2. Init Repo
	if _, err := svc.Run(tempDir, "init"); err != nil {
		log.Fatalf("Failed to git init: %v", err)
	}

	// Config user for commit
	svc.Run(tempDir, "config", "user.email", "test@juki.app")
	svc.Run(tempDir, "config", "user.name", "Juki Test")

	// 3. Create initial file
	readmePath := filepath.Join(tempDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Initial"), 0644); err != nil {
		log.Fatalf("Failed to write file: %v", err)
	}

	// 4. Commit
	if err := svc.Commit(tempDir, "Initial commit"); err != nil {
		log.Fatalf("Failed to commit: %v", err)
	}
	fmt.Println("✅ Initial commit passed")

	// 5. Start Session
	sessionID := "test-session-123"
	branch, err := svc.StartSession(tempDir, sessionID)
	if err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}
	fmt.Printf("✅ Started session branch: %s\n", branch)

	// 6. Modify file
	if err := os.WriteFile(readmePath, []byte("# Initial\nUpdated in session"), 0644); err != nil {
		log.Fatalf("Failed to update file: %v", err)
	}

	// 7. Commit in session
	if err := svc.Commit(tempDir, "Session update"); err != nil {
		log.Fatalf("Failed to session commit: %v", err)
	}
	fmt.Println("✅ Session commit passed")

	// 8. Publish Session
	// Switch back to master first so juki-dev (if created) branches from master, not session
	svc.Checkout(tempDir, "master", false)

	if err := svc.PublishSession(tempDir, branch); err != nil {
		log.Fatalf("Failed to publish session: %v", err)
	}
	fmt.Println("✅ Publish session passed")

	// Verify we are on juki-dev
	current, _ := svc.CurrentBranch(tempDir)
	if current != "juki-dev" {
		log.Fatalf("Expected branch juki-dev, got %s", current)
	}
	fmt.Println("✅ Validated final branch is juki-dev")
}
