package services

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"juki-engine/pkg/data/models"
	"juki-engine/pkg/services/marketplace"

	"github.com/google/uuid"
)

// InstallMarketplaceEntry resolves an entry's install_type and executes it:
//
//   - "copy":  download the source file -> parse via bridge -> register as a
//     UserComponent for the project
//   - "npm":   install the npm package via the bridge's plugin installer
//   - "patch": download an AST patch spec and apply it (delegates to the
//     bridge's `compose`/AST tooling — stubbed pending ts-morph patch format)
//
// Records a MarketplaceInstall row on success so ListInstalled can report it
// and duplicate installs can be detected.
func (s *service) InstallMarketplaceEntry(projectID string, entry *marketplace.Entry) (string, error) {
	if entry == nil {
		return "", fmt.Errorf("entry is nil")
	}

	// Skip if already installed.
	if existing, err := s.repository.GetMarketplaceInstall(projectID, entry.ID); err == nil && existing != nil {
		return existing.InstalledRef, nil
	}

	var ref string
	var err error

	switch entry.InstallType {
	case "copy":
		ref, err = s.installByCopy(projectID, entry)
	case "npm":
		ref, err = s.installByNpm(projectID, entry)
	case "patch":
		ref, err = s.installByPatch(projectID, entry)
	default:
		return "", fmt.Errorf("unknown install_type %q for entry %s", entry.InstallType, entry.ID)
	}

	if err != nil {
		return "", err
	}

	record := &models.MarketplaceInstall{
		ID:           uuid.NewString(),
		ProjectID:    projectID,
		EntryID:      entry.ID,
		Name:         entry.Name,
		Type:         entry.Type,
		InstalledRef: ref,
	}
	if cerr := s.repository.CreateMarketplaceInstall(record); cerr != nil {
		// Installation itself succeeded — log but don't fail the whole call.
		fmt.Printf("[Marketplace] warning: failed to record install for %s: %v\n", entry.ID, cerr)
	}

	return ref, nil
}

func (s *service) installByCopy(projectID string, entry *marketplace.Entry) (string, error) {
	if entry.Source == "" {
		return "", fmt.Errorf("entry %s has install_type=copy but no source URL", entry.ID)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(entry.Source)
	if err != nil {
		return "", fmt.Errorf("failed to download %s: %w", entry.Source, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download of %s returned status %d", entry.Source, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read downloaded source: %w", err)
	}

	contentJSON := "[]"
	if s.bridge != nil {
		if parsed, perr := s.bridge.Call("parse_jsx", map[string]interface{}{"code": string(body)}); perr == nil {
			contentJSON = string(parsed)
		}
	}

	component := &models.UserComponent{
		ID:        uuid.NewString(),
		ProjectID: projectID,
		Name:      entry.Name,
		Content:   contentJSON,
	}
	if err := s.repository.CreateComponent(component); err != nil {
		return "", fmt.Errorf("failed to register component: %w", err)
	}

	return component.ID, nil
}

func (s *service) installByNpm(projectID string, entry *marketplace.Entry) (string, error) {
	if entry.Package == "" {
		return "", fmt.Errorf("entry %s has install_type=npm but no package name", entry.ID)
	}
	if s.bridge == nil {
		return "", fmt.Errorf("bridge is not initialized")
	}

	if _, err := s.bridge.Call("install_plugin", map[string]string{
		"project_id":  projectID,
		"plugin_name": entry.Package,
		"version":     entry.Version,
	}); err != nil {
		return "", fmt.Errorf("failed to install package %s: %w", entry.Package, err)
	}

	return entry.Package, nil
}

func (s *service) installByPatch(projectID string, entry *marketplace.Entry) (string, error) {
	if entry.Source == "" {
		return "", fmt.Errorf("entry %s has install_type=patch but no patch spec URL", entry.ID)
	}
	if s.bridge == nil {
		return "", fmt.Errorf("bridge is not initialized")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(entry.Source)
	if err != nil {
		return "", fmt.Errorf("failed to download patch spec %s: %w", entry.Source, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read patch spec: %w", err)
	}

	if _, err := s.bridge.Call("apply_patch", map[string]interface{}{
		"project_id": projectID,
		"spec":       string(body),
	}); err != nil {
		return "", fmt.Errorf("failed to apply AST patch for %s: %w", entry.ID, err)
	}

	return entry.ID, nil
}

// ListInstalledMarketplaceEntries returns the install records for a project.
func (s *service) ListInstalledMarketplaceEntries(projectID string) ([]*models.MarketplaceInstall, error) {
	return s.repository.ListMarketplaceInstalls(projectID)
}
