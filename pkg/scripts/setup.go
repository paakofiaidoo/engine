package scripts

import (
	"encoding/json"
	"fmt"
	"juki-engine/data/dtos"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

type Scripts interface {
	CreateAppNextApp(project dtos.Project)
}

type scripts struct {
}

type JukiConfig struct {
	PackageManager string `json:"packageManager"`
}

func (s scripts) CreateAppNextApp(app dtos.Project) {
	log.Printf("[Script] CreateAppNextApp: Checking Node.js...\n")
	// Check if node is installed
	if _, err := exec.LookPath("node"); err != nil {
		log.Println("[Script] Error: Node.js is not installed")
		return
	}

	// Determine package manager
	pm := s.getPreferredPackageManager()
	log.Printf("[Script] CreateAppNextApp: Selected Package Manager: %s\n", pm)

	// Run npx create-next-app
	// We use the project path as the target directory

	log.Printf("[Script] CreateAppNextApp: Target Path: %s\n", app.Path)

	// Use 'yes' to automatically answer prompts (like "Would you like to use React Compiler?")
	// We wrap in sh -c to use the pipe.
	// Also added --no-turbopack to avoid that prompt.
	// Added --use-<pm> to specify package manager.
	cmdStr := fmt.Sprintf("yes | npx create-next-app@latest %s --typescript --eslint --tailwind --no-src-dir --app --import-alias @/* --no-turbopack --use-%s", app.Path, pm)
	cmd := exec.Command("sh", "-c", cmdStr)

	// Stream output to stdout/stderr for now
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("[Script] Running: %s\n", cmdStr)
	if err := cmd.Run(); err != nil {
		log.Printf("[Script] Error creating Next.js app: %v\n", err)
	} else {
		log.Println("[Script] Next.js app created successfully!")
	}
}

func (s scripts) getPreferredPackageManager() string {
	// 1. Check juki.config.json in workspace root
	// Assuming engine is in .juki/engine, workspace root is ../../
	cwd, err := os.Getwd()
	if err == nil {
		configPath := filepath.Join(cwd, "..", "..", "juki.config.json")
		if data, err := os.ReadFile(configPath); err == nil {
			var config JukiConfig
			if err := json.Unmarshal(data, &config); err == nil && config.PackageManager != "" {
				return config.PackageManager
			}
		}
	}

	// 2. Check available package managers in order: pnpm > yarn > npm > bun
	if _, err := exec.LookPath("pnpm"); err == nil {
		return "pnpm"
	}
	if _, err := exec.LookPath("yarn"); err == nil {
		return "yarn"
	}
	if _, err := exec.LookPath("npm"); err == nil {
		return "npm"
	}
	if _, err := exec.LookPath("bun"); err == nil {
		return "bun"
	}

	return "npm" // Fallback
}

func NewScript() Scripts {
	return &scripts{}
}
