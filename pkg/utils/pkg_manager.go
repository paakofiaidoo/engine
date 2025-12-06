package utils

import (
	"os"
	"path/filepath"
)

type PackageManager string

const (
	NPM  PackageManager = "npm"
	YARN PackageManager = "yarn"
	PNPM PackageManager = "pnpm"
	BUN  PackageManager = "bun"
)

func DetectPackageManager(projectDir string) (PackageManager, error) {
	if _, err := os.Stat(filepath.Join(projectDir, "pnpm-lock.yaml")); err == nil {
		return PNPM, nil
	}
	if _, err := os.Stat(filepath.Join(projectDir, "yarn.lock")); err == nil {
		return YARN, nil
	}
	if _, err := os.Stat(filepath.Join(projectDir, "bun.lockb")); err == nil {
		return BUN, nil
	}
	if _, err := os.Stat(filepath.Join(projectDir, "package-lock.json")); err == nil {
		return NPM, nil
	}
	// Default to npm if no lockfile found, or maybe error?
	// Let's default to npm but log a warning ideally.
	return NPM, nil
}

func (pm PackageManager) InstallCommand() []string {
	switch pm {
	case YARN:
		return []string{"yarn", "install"}
	case PNPM:
		return []string{"pnpm", "install"}
	case BUN:
		return []string{"bun", "install"}
	default:
		return []string{"npm", "install"}
	}
}

func (pm PackageManager) AddCommand(packages ...string) []string {
	switch pm {
	case YARN:
		return append([]string{"yarn", "add"}, packages...)
	case PNPM:
		return append([]string{"pnpm", "add"}, packages...)
	case BUN:
		return append([]string{"bun", "add"}, packages...)
	default:
		return append([]string{"npm", "install"}, packages...)
	}
}
