package services

import (
	"errors"
	"fmt"
	"juki-engine/pkg/data/dtos"
	"juki-engine/pkg/data/models"
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
		// Default to ../../projects/<project_name> relative to engine (.juki/engine)
		defaultPath := filepath.Join(cwd, "..", "..", "projects", project.Name)
		absPath, err := filepath.Abs(defaultPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
		}
		project.Path = absPath
		log.Printf("[Service] CreateProject: Path was empty, defaulting to: %s\n", project.Path)
	}

	// 1. Determine Port
	maxPort, err := s.repository.GetMaxPort()
	if err != nil {
		log.Printf("[Service] CreateProject: Failed to get max port, defaulting to 5175: %v\n", err)
		maxPort = 5175
	}
	if maxPort == 0 {
		maxPort = 5175 // Start base
	}
	newPort := maxPort + 1
	log.Printf("[Service] CreateProject: Assigning Port %d\n", newPort)

	// 2. Create DB Record
	newProject := &models.Project{
		Name:        project.Name,
		Path:        project.Path,
		Framework:   project.Framework,
		Description: project.Description,
		Port:        newPort,
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

	var newPages []models.Page
	var newLayouts []models.Layout

	err = filepath.Walk(appDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Removed duplicate error check: if err != nil { return err }

		relPath, _ := filepath.Rel(appDir, path)
		dir := filepath.Dir(relPath)
		route := "/"
		if dir != "." {
			route = "/" + dir
		}

		// Handle Page
		if !info.IsDir() && info.Name() == "page.tsx" {
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			// Check DB
			existingPage := -1
			for i, p := range project.Pages {
				if p.Route == route {
					existingPage = i
					break
				}
			}
			if existingPage != -1 {
				project.Pages[existingPage].RawContent = string(content)
			} else {
				newPage := models.Page{
					ID:         uuid.NewString(),
					ProjectID:  project.ID,
					Name:       dir,
					Route:      route,
					RawContent: string(content),
					Content:    "[]",
				}
				if dir == "." {
					newPage.Name = "Home"
				}
				newPages = append(newPages, newPage)
			}
		}

		// Handle Layout
		if !info.IsDir() && info.Name() == "layout.tsx" {
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			// Check DB
			existingLayout := -1
			for i, l := range project.Layouts {
				if l.Route == route {
					existingLayout = i
					break
				}
			}

			if existingLayout != -1 {
				project.Layouts[existingLayout].RawContent = string(content)
			} else {
				newLayout := models.Layout{
					ID:         uuid.NewString(),
					ProjectID:  project.ID,
					Name:       dir + " Layout",
					Route:      route,
					RawContent: string(content),
					Content:    "[]",
					IsRoot:     route == "/",
				}
				if route == "/" {
					newLayout.Name = "Root Layout"
				}
				newLayouts = append(newLayouts, newLayout)
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

	// 4. Save updates to DB (Project + Existing Items)
	// We need to save the project to persist RawContent updates for existing pages/layouts
	if err := s.repository.UpdateProject(project); err != nil {
		return nil, err
	}

	// 5. Create new items
	for _, p := range newPages {
		if err := s.repository.CreatePage(&p); err != nil {
			log.Printf("Failed to create page %s: %v\n", p.Name, err)
		} else {
			// Add to project for return
			project.Pages = append(project.Pages, p)
		}
	}
	for _, l := range newLayouts {
		if err := s.repository.CreateLayout(&l); err != nil { // Assuming CreateLayout exists in repository
			log.Printf("Failed to create layout %s: %v\n", l.Name, err)
		} else {
			// Add to project for return
			project.Layouts = append(project.Layouts, l)
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

	// 7. Build Route Tree
	rootRoute := s.buildRouteTree(project.Pages, project.Layouts)

	return &dtos.Project{
		ID:          project.ID,
		Name:        project.Name,
		Path:        project.Path,
		Framework:   project.Framework,
		Description: project.Description,
		ApiKey:      project.ApiKey,
		GlobalCSS:   globalCSSContent,
		Pages:       dtosPages,
		RootRoute:   rootRoute,
		Port:        project.Port,
	}, nil
}

// buildRouteTree constructs a proto-compatible RouteNode tree from a flat list of pages and layouts
func (s *service) buildRouteTree(pages []models.Page, layouts []models.Layout) dtos.RouteNode {
	// A simple implementation that assumes standard Next.js routing
	// This would ideally be more robust and handle dynamic segments/layouts
	root := dtos.RouteNode{
		ID:       "root",
		Name:     "Root",
		Segment:  "/",
		FullPath: "/",
		Type:     "STATIC",
		Children: []dtos.RouteNode{},
	}

	// Helper to find layout for a specific route
	findLayout := func(route string) string {
		for _, l := range layouts {
			if l.Route == route {
				return l.ID
			}
		}
		return ""
	}

	root.LayoutID = findLayout("/")

	for _, page := range pages {
		if page.Route == "/" {
			root.PageID = page.ID
			continue
		}

		// Split route into segments (e.g. "/blog/post" -> ["blog", "post"])
		segments := strings.Split(strings.TrimPrefix(page.Route, "/"), "/")
		currentNode := &root

		// Traverse/Build tree
		pathSoFar := ""
		for _, segment := range segments {
			pathSoFar += "/" + segment

			// Check if child exists
			var child *dtos.RouteNode
			for i := range currentNode.Children {
				if currentNode.Children[i].Segment == segment {
					child = &currentNode.Children[i]
					break
				}
			}

			// If not, create it
			if child == nil {
				newNode := dtos.RouteNode{
					ID:       uuid.NewString(),
					Name:     segment, // formatting could be better
					Segment:  segment,
					FullPath: pathSoFar,
					Type:     "STATIC", // Default
					Children: []dtos.RouteNode{},
				}
				// Detect dynamic segments
				if strings.HasPrefix(segment, "[") && strings.HasSuffix(segment, "]") {
					newNode.Type = "DYNAMIC"
				}

				// Assign Layout if exists for this exact path
				newNode.LayoutID = findLayout(pathSoFar)

				currentNode.Children = append(currentNode.Children, newNode)
				// Re-point child to the newly added node (last index)
				// Note: In Go, we need to be careful with pointers to slice elements if slice reallocates.
				// For this simple logic, we'll re-fetch the pointer or use index.
				child = &currentNode.Children[len(currentNode.Children)-1]
			}

			// If this is the last segment, assign the PageID
			if pathSoFar == page.Route {
				child.PageID = page.ID
			}

			currentNode = child
		}
	}
	return root
}

func (s *service) RunProject(projectID string) (int, error) {
	log.Printf("[Service] RunProject: Running project %s\n", projectID)

	project, err := s.repository.GetProject(projectID)
	if err != nil {
		return 0, err
	}

	// Script path relative to engine cwd (.juki/engine) -> ../../scripts/run-project.sh
	cwd, err := os.Getwd()
	if err != nil {
		log.Printf("[Service] RunProject: Failed to get CWD: %v\n", err)
		return 0, err
	}
	scriptPath := filepath.Join(cwd, "..", "..", "scripts", "run-project.sh")

	// Execute Run Script
	cmd := exec.Command(scriptPath, project.Path, fmt.Sprintf("%d", project.Port))
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[Service] RunProject: Failed for %s: %v\nOutput: %s\n", project.Name, err, string(output))
		// Optional: return error, or maybe just proceed if it was "already running" exit code 0 handled by script?
		// Script returns exit 0 if already running.
		// If real error, script returns 1.
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() != 0 {
				return 0, fmt.Errorf("failed to run project: %s", string(output))
			}
		} else {
			return 0, err
		}
	} else {
		log.Printf("[Service] RunProject: Triggered for %s (Port %d)\n", project.Name, project.Port)
	}

	return project.Port, nil
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
