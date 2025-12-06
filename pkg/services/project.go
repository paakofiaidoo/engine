package services

import (
	"errors"
	"fmt"
	"juki-engine/data/dtos"
	"juki-engine/data/models"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func (s *service) CreateProject(project dtos.Project) (*models.Project, error) {

	log.Printf("[Service] CreateProject: Starting for %s (Path: %s)\n", project.Name, project.Path)

	// 0. Set Default Path if empty
	if project.Path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
		// Default to ../../<project_name> relative to engine (.juki/engine)
		// This puts it in the root of the workspace (juki-builder)
		defaultPath := filepath.Join(cwd, "..", "..", project.Name)
		absPath, err := filepath.Abs(defaultPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
		}
		project.Path = absPath
		log.Printf("[Service] CreateProject: Path was empty, defaulting to: %s\n", project.Path)
	}

	// 1. Create DB Record
	newProject := &models.Project{
		Name:        project.Name,
		Path:        project.Path,
		Framework:   project.Framework,
		Description: project.Description,
	}

	if err := s.repository.CreateProject(newProject); err != nil {
		log.Printf("[Service] CreateProject: DB Insert Failed: %v\n", err)
		return nil, fmt.Errorf("failed to create project record: %w", err)
	}
	log.Printf("[Service] CreateProject: DB Record Created (ID: %s)\n", newProject.ID)

	// 2. Run Script
	log.Printf("[Service] CreateProject: Running Scaffolding Script for %s...\n", project.Framework)
	switch project.Framework {
	case "NextJS":
		s.scripts.CreateAppNextApp(project)
	default:
		log.Printf("[Service] CreateProject: Unsupported Framework: %s\n", project.Framework)
		// If script fails or framework not supported, should we delete the DB record?
		// For now, let's just return error.
		return nil, errors.New("not supported framework")

	}

	log.Println("[Service] CreateProject: Completed Successfully")

	// 3. Create Default Home Page
	// Since we can't easily parse the generated page.tsx yet, we'll create a default Juki page
	// that overwrites the visual representation (but not the file yet, until save).
	// Ideally, we should parse the file. For now, we provide a starting point.
	defaultPage := &models.Page{
		ID:          uuid.NewString(),
		ProjectID:   newProject.ID,
		Name:        "Home",
		Description: "Main landing page",
		Route:       "/",
		Content:     `[{"id":"root","name":"Root","type":"ELEMENT","tag":"div","props":{"className":"min-h-screen p-8"},"content":[{"id":"h1","name":"Title","type":"ELEMENT","tag":"h1","props":{"className":"text-4xl font-bold mb-4"},"content":"Welcome to Juki"}]}]`,
	}

	if err := s.repository.CreatePage(defaultPage); err != nil {
		log.Printf("[Service] CreateProject: Failed to create default page: %v\n", err)
		// Don't fail the whole project creation for this
	} else {
		log.Printf("[Service] CreateProject: Default Home Page Created (ID: %s)\n", defaultPage.ID)
	}

	return newProject, nil
}

func (s *service) ListProjects() ([]dtos.Project, error) {
	log.Println("[Service] ListProjects: Fetching all projects...")
	projects, err := s.repository.GetProjects()
	if err != nil {
		log.Printf("[Service] ListProjects: Failed to fetch projects: %v\n", err)
		return nil, err
	}
	log.Printf("[Service] ListProjects: Found %d projects\n", len(projects))

	var dtosProjects []dtos.Project
	for _, p := range projects {
		var pages []dtos.PageDto
		for _, page := range p.Pages {
			pages = append(pages, dtos.PageDto{
				ID:         page.ID,
				Name:       page.Name,
				Route:      page.Route,
				Content:    page.Content,
				RawContent: page.RawContent,
			})
		}
		dtosProjects = append(dtosProjects, dtos.Project{
			ID:          p.ID,
			Name:        p.Name,
			Path:        p.Path,
			Framework:   p.Framework,
			Description: p.Description,
			ApiKey:      p.ApiKey,
			Pages:       pages,
		})
	}
	return dtosProjects, nil
}

func (s *service) GetProject(id string) (*models.Project, error) {
	return s.repository.GetProject(id)
}

func (s *service) DeleteProject(id string) error {
	log.Printf("[Service] DeleteProject: Deleting project %s\n", id)

	// 1. Get Project to find path
	project, err := s.repository.GetProject(id)
	if err != nil {
		log.Printf("[Service] DeleteProject: Failed to find project: %v\n", err)
		return err
	}

	// 2. Delete from DB
	if err := s.repository.DeleteProject(id); err != nil {
		log.Printf("[Service] DeleteProject: Failed to delete project from DB: %v\n", err)
		return err
	}

	// 3. Delete from Filesystem
	if project.Path != "" {
		log.Printf("[Service] DeleteProject: Removing directory %s\n", project.Path)
		if err := os.RemoveAll(project.Path); err != nil {
			log.Printf("[Service] DeleteProject: Failed to remove directory: %v\n", err)
			// We don't return error here because DB delete was successful,
			// and we don't want to block the user if file permission issues exist.
		}
	}

	log.Println("[Service] DeleteProject: Success")
	return nil
}

func (s *service) UpdateProject(project dtos.Project) error {
	log.Printf("[Service] UpdateProject: Updating project %s\n", project.ID)

	// 1. Get existing project
	existingProject, err := s.repository.GetProject(project.ID)
	if err != nil {
		return err
	}

	// 2. Update fields
	existingProject.Name = project.Name
	existingProject.Description = project.Description
	existingProject.ApiKey = project.ApiKey
	// Path and Framework are generally immutable after creation for now

	// 3. Save to DB
	if err := s.repository.UpdateProject(existingProject); err != nil {
		log.Printf("[Service] UpdateProject: Failed to update DB: %v\n", err)
		return err
	}

	log.Println("[Service] UpdateProject: Success")
	return nil
}

func (s *service) GetProjectById(id string) error {
	_, err := s.repository.GetProject(id)
	return err
}

func (s *service) SyncProject(id string) (*dtos.Project, error) {
	log.Printf("[Service] SyncProject: Syncing project %s\n", id)

	// 1. Get Project
	project, err := s.repository.GetProject(id)
	if err != nil {
		return nil, err
	}

	if project.Path == "" {
		return nil, fmt.Errorf("project path is empty")
	}

	// 2. Walk app directory
	appDir := filepath.Join(project.Path, "app")
	log.Printf("[Service] SyncProject: Scanning %s\n", appDir)

	var pages []models.Page

	err = filepath.Walk(appDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "page.tsx" {
			// Found a page
			relPath, _ := filepath.Rel(appDir, path)
			dir := filepath.Dir(relPath)
			route := "/"
			if dir != "." {
				route = "/" + dir
			}

			content, err := os.ReadFile(path)
			if err != nil {
				log.Printf("Failed to read file %s: %v\n", path, err)
				return nil
			}

			log.Printf("Found page: %s (Route: %s)\n", relPath, route)

			// Check if page exists in DB
			existingPage := -1
			for i, p := range project.Pages {
				if p.Route == route {
					existingPage = i
					break
				}
			}

			if existingPage != -1 {
				// Update existing
				project.Pages[existingPage].RawContent = string(content)
				// We DO NOT update Content (JSON) here, frontend will do it
				// Or we could try to parse it here if we had a parser? No, frontend does it.
				// But we need to save RawContent to DB.
				// Assuming Repository has UpdatePage or SaveProject cascades.
				// GORM Save should cascade if configured.
			} else {
				// Create new page record
				newPage := models.Page{
					ID:         uuid.NewString(),
					ProjectID:  project.ID,
					Name:       dir, // Use dir name as page name
					Route:      route,
					RawContent: string(content),
					Content:    "[]", // Empty JSON for now
				}
				if dir == "." {
					newPage.Name = "Home"
				}
				pages = append(pages, newPage)
			}
		}
		return nil
	})

	// 3. Read globals.css
	var globalCSSContent string
	globalCssPath := filepath.Join(appDir, "globals.css")
	if _, err := os.Stat(globalCssPath); err == nil {
		content, err := os.ReadFile(globalCssPath)
		if err == nil {
			globalCSSContent = string(content)
			log.Printf("[Service] SyncProject: Found globals.css (%d bytes)\n", len(content))
		}
	}

	// 4. Save updates to DB
	// We need to save the project to persist RawContent updates
	if err := s.repository.UpdateProject(project); err != nil {
		return nil, err
	}

	// 5. Create new pages
	for _, p := range pages {
		if err := s.repository.CreatePage(&p); err != nil {
			log.Printf("Failed to create page %s: %v\n", p.Name, err)
		} else {
			// Add to project for return
			project.Pages = append(project.Pages, p)
		}
	}

	// 6. Return DTO
	var dtosPages []dtos.PageDto
	for _, page := range project.Pages {
		dtosPages = append(dtosPages, dtos.PageDto{
			ID:         page.ID,
			Name:       page.Name,
			Route:      page.Route,
			Content:    page.Content,
			RawContent: page.RawContent,
		})
	}

	return &dtos.Project{
		ID:          project.ID,
		Name:        project.Name,
		Path:        project.Path,
		Framework:   project.Framework,
		Description: project.Description,
		ApiKey:      project.ApiKey,
		GlobalCSS:   globalCSSContent,
		Pages:       dtosPages,
	}, nil
}

func (s *service) SavePage(projectID string, pageID string, content string) error {
	log.Printf("[Service] SavePage: Saving page %s for project %s\n", pageID, projectID)

	project, err := s.repository.GetProject(projectID)
	if err != nil {
		return err
	}

	page, err := s.repository.GetPage(pageID)
	if err != nil {
		return err
	}

	// Construct file path
	// Assuming Next.js App Router: project/app/[route]/page.tsx
	// Route "/" -> "app/page.tsx"
	// Route "/about" -> "app/about/page.tsx"

	relPath := page.Route
	if relPath == "/" {
		relPath = ""
	} else if strings.HasPrefix(relPath, "/") {
		relPath = relPath[1:]
	}

	filePath := filepath.Join(project.Path, "app", relPath, "page.tsx")
	log.Printf("[Service] SavePage: Writing to %s\n", filePath)

	// Write to file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Update DB
	page.RawContent = content
	// We might want to update the JSON content too if we trust the frontend sent it correctly?
	// For now, let's assume the frontend syncs state separately or we just care about the file.

	if err := s.repository.UpdatePage(page); err != nil {
		log.Printf("Failed to update page in DB: %v\n", err)
		// Non-fatal if file write succeeded
	}

	return nil
}

func (s *service) BuildProject(projectID string) (string, error) {
	log.Printf("[Service] BuildProject: Building project %s\n", projectID)

	project, err := s.repository.GetProject(projectID)
	if err != nil {
		return "", err
	}

	// Run npm run build
	cmd := exec.Command("npm", "run", "build")
	cmd.Dir = project.Path

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("build failed: %w\nOutput:\n%s", err, string(output))
	}

	return string(output), nil
}
