package marketplace

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Entry mirrors a manifest entry in the GitHub registry's registry.json index.
// JSON tags match the documented manifest format (see Part 6 of the feature plan).
type Entry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Type        string   `json:"type"`         // "template" | "component" | "plugin" | "icon_pack"
	InstallType string   `json:"install_type"` // "copy" | "npm" | "patch"
	Source      string   `json:"source"`
	Package     string   `json:"package"`
	Preview     string   `json:"preview"`
	Tags        []string `json:"tags"`
}

type registryIndex struct {
	Entries []Entry `json:"entries"`
}

const (
	defaultRegistryURL = "https://raw.githubusercontent.com/juki-builder/marketplace/main/registry.json"
	cacheTTL           = 1 * time.Hour
)

// Service fetches and caches the marketplace registry (GitHub-as-DB pattern),
// serving search/list/install over the engine's gRPC layer.
//
// Cache is held in memory and persisted to ~/.juki/registry-cache.json so a
// cold engine restart doesn't require an immediate network round-trip.
type Service struct {
	mu          sync.RWMutex
	registryURL string
	cachePath   string
	entries     []Entry
	fetchedAt   time.Time
}

func NewService() *Service {
	home, _ := os.UserHomeDir()
	cachePath := filepath.Join(home, ".juki", "registry-cache.json")

	url := os.Getenv("JUKI_MARKETPLACE_REGISTRY_URL")
	if url == "" {
		url = defaultRegistryURL
	}

	s := &Service{registryURL: url, cachePath: cachePath}
	s.loadFromDisk()
	return s
}

type diskCache struct {
	FetchedAt int64   `json:"fetched_at_ms"`
	Entries   []Entry `json:"entries"`
}

func (s *Service) loadFromDisk() {
	data, err := os.ReadFile(s.cachePath)
	if err != nil {
		return
	}
	var dc diskCache
	if err := json.Unmarshal(data, &dc); err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = dc.Entries
	s.fetchedAt = time.UnixMilli(dc.FetchedAt)
}

func (s *Service) saveToDisk() {
	s.mu.RLock()
	dc := diskCache{FetchedAt: s.fetchedAt.UnixMilli(), Entries: s.entries}
	s.mu.RUnlock()

	data, err := json.Marshal(dc)
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(s.cachePath), 0o755)
	_ = os.WriteFile(s.cachePath, data, 0o644)
}

// Refresh fetches the registry index from GitHub if the cache is stale (or `force`
// is set), otherwise returns the cached entries. Returns (entryCount, fetchedAtMs, fromCache).
func (s *Service) Refresh(force bool) (int, int64, bool, error) {
	s.mu.RLock()
	fresh := time.Since(s.fetchedAt) < cacheTTL && len(s.entries) > 0
	s.mu.RUnlock()

	if fresh && !force {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return len(s.entries), s.fetchedAt.UnixMilli(), true, nil
	}

	entries, err := s.fetchRemote()
	if err != nil {
		// Fall back to whatever is cached (even if stale) rather than failing hard —
		// keeps the marketplace usable offline / when GitHub rate-limits us.
		s.mu.RLock()
		count := len(s.entries)
		fetchedAt := s.fetchedAt.UnixMilli()
		s.mu.RUnlock()
		if count > 0 {
			return count, fetchedAt, true, nil
		}
		return 0, 0, false, err
	}

	now := time.Now()
	s.mu.Lock()
	s.entries = entries
	s.fetchedAt = now
	s.mu.Unlock()

	s.saveToDisk()

	return len(entries), now.UnixMilli(), false, nil
}

func (s *Service) fetchRemote() ([]Entry, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(s.registryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry fetch returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry response: %w", err)
	}

	var idx registryIndex
	if err := json.Unmarshal(body, &idx); err != nil {
		return nil, fmt.Errorf("failed to parse registry.json: %w", err)
	}

	return idx.Entries, nil
}

// ensureLoaded lazily fetches the registry on first use so List/Get work even
// if RefreshRegistry was never explicitly called.
func (s *Service) ensureLoaded() {
	s.mu.RLock()
	empty := len(s.entries) == 0
	s.mu.RUnlock()
	if empty {
		_, _, _, _ = s.Refresh(false)
	}
}

// List returns entries optionally filtered by type and a case-insensitive query
// matched against name, description, and tags.
func (s *Service) List(entryType, query string) []Entry {
	s.ensureLoaded()

	s.mu.RLock()
	defer s.mu.RUnlock()

	q := strings.ToLower(strings.TrimSpace(query))
	var out []Entry
	for _, e := range s.entries {
		if entryType != "" && entryType != "unspecified" && e.Type != entryType {
			continue
		}
		if q != "" && !matches(e, q) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func matches(e Entry, q string) bool {
	if strings.Contains(strings.ToLower(e.Name), q) || strings.Contains(strings.ToLower(e.Description), q) {
		return true
	}
	for _, t := range e.Tags {
		if strings.Contains(strings.ToLower(t), q) {
			return true
		}
	}
	return false
}

// Get returns a single entry by id.
func (s *Service) Get(id string) (*Entry, bool) {
	s.ensureLoaded()

	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.entries {
		if s.entries[i].ID == id {
			e := s.entries[i]
			return &e, true
		}
	}
	return nil, false
}
