package system

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
)

// EnsureDependencies checks for required tools and installs them if missing.
// Targeted for macOS (Darwin) using Homebrew.
func EnsureDependencies() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("automatic dependency installation is only supported on macOS")
	}

	log.Println("Checking system dependencies...")

	// 1. Check Homebrew
	if _, err := exec.LookPath("brew"); err != nil {
		log.Println("Homebrew not found. Installing...")
		cmd := exec.Command("/bin/bash", "-c", "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin // Interactive? Might fail if password needed.
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install Homebrew: %w", err)
		}
		// Add brew to PATH for this session if needed (standard locations)
		os.Setenv("PATH", os.Getenv("PATH")+":/opt/homebrew/bin:/usr/local/bin")
	}

	// 2. Check Go
	if _, err := exec.LookPath("go"); err != nil {
		if err := installWithBrew("go"); err != nil {
			return err
		}
	}

	// 3. Check Node
	if _, err := exec.LookPath("node"); err != nil {
		if err := installWithBrew("node"); err != nil {
			return err
		}
	}

	// 4. Check pnpm
	if _, err := exec.LookPath("pnpm"); err != nil {
		if err := installWithBrew("pnpm"); err != nil {
			return err
		}
	}

	log.Println("All system dependencies are satisfied.")
	return nil
}

func installWithBrew(pkg string) error {
	log.Printf("Installing %s via Homebrew...", pkg)
	cmd := exec.Command("brew", "install", pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
