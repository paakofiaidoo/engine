package api

import (
	"context"
	"fmt"
	dtos2 "juki-engine/pkg/data/dtos"

	enginev1 "juki-engine/pkg/gen/juki/engine/v1"
	"juki-engine/pkg/services"

	"connectrpc.com/connect"
	"github.com/fsnotify/fsnotify"
)

type EngineServer struct {
	svc services.Service
}

func NewEngineServer(svc services.Service) *EngineServer {
	return &EngineServer{svc: svc}
}

func (s *EngineServer) Ping(
	ctx context.Context,
	req *connect.Request[enginev1.PingRequest],
) (*connect.Response[enginev1.PingResponse], error) {
	workerMsg, err := s.svc.PingWorker()
	msg := fmt.Sprintf("Pong: %s", req.Msg.Message)
	if err == nil {
		msg += fmt.Sprintf(" | Worker: %s", workerMsg)
	} else {
		msg += fmt.Sprintf(" | Worker Error: %v", err)
	}

	return connect.NewResponse(&enginev1.PingResponse{
		Message: msg,
	}), nil
}

// Project Handlers

func (s *EngineServer) CreateProject(
	ctx context.Context,
	req *connect.Request[enginev1.CreateProjectRequest],
) (*connect.Response[enginev1.CreateProjectResponse], error) {
	fmt.Printf("[API] CreateProject Request Received: Name=%s, Path=%s, Framework=%s\n", req.Msg.Name, req.Msg.Path, req.Msg.Framework)

	projectDTO := dtos2.Project{
		Name:        req.Msg.Name,
		Path:        req.Msg.Path,
		Framework:   req.Msg.Framework,
		Description: req.Msg.Description,
	}

	createdProject, err := s.svc.CreateProject(projectDTO)
	if err != nil {
		fmt.Printf("[API] CreateProject Failed: %v\n", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	fmt.Printf("[API] CreateProject Success: ID=%s\n", createdProject.ID)

	return connect.NewResponse(&enginev1.CreateProjectResponse{
		Project: &enginev1.Project{
			Id:          createdProject.ID,
			Name:        createdProject.Name,
			Path:        createdProject.Path,
			Settings:    &enginev1.ProjectSettings{Framework: createdProject.Framework},
			Description: createdProject.Description,
		},
	}), nil
}

func (s *EngineServer) GetProject(
	ctx context.Context,
	req *connect.Request[enginev1.GetProjectRequest],
) (*connect.Response[enginev1.GetProjectResponse], error) {
	fmt.Printf("[API] GetProject Request Received: ID=%s\n", req.Msg.Id)

	project, err := s.svc.GetProject(req.Msg.Id)
	if err != nil {
		fmt.Printf("[API] GetProject Failed: %v\n", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	var pbPages []*enginev1.Page
	for _, page := range project.Pages {
		pbPages = append(pbPages, &enginev1.Page{
			Id:      page.ID,
			Name:    page.Name,
			Route:   page.Route,
			Content: page.Content,
		})
	}

	return connect.NewResponse(&enginev1.GetProjectResponse{
		Project: &enginev1.Project{
			Id:          project.ID,
			Name:        project.Name,
			Path:        project.Path,
			Settings:    &enginev1.ProjectSettings{Framework: project.Framework},
			Description: project.Description,
			ApiKey:      project.ApiKey,
			Pages:       pbPages,
		},
	}), nil
}

func (s *EngineServer) ListProjects(
	ctx context.Context,
	req *connect.Request[enginev1.ListProjectsRequest],
) (*connect.Response[enginev1.ListProjectsResponse], error) {
	fmt.Println("[API] ListProjects Request Received")

	projects, err := s.svc.ListProjects()
	if err != nil {
		fmt.Printf("[API] ListProjects Failed: %v\n", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var pbProjects []*enginev1.Project
	for _, p := range projects {
		var pbPages []*enginev1.Page
		for _, page := range p.Pages {
			pbPages = append(pbPages, &enginev1.Page{
				Id:      page.ID,
				Name:    page.Name,
				Route:   page.Route,
				Content: page.Content,
			})
		}

		pbProjects = append(pbProjects, &enginev1.Project{
			Id:          p.ID,
			Name:        p.Name,
			Path:        p.Path,
			Settings:    &enginev1.ProjectSettings{Framework: p.Framework},
			Description: p.Description,
			ApiKey:      p.ApiKey,
			Pages:       pbPages,
		})
	}

	fmt.Printf("[API] ListProjects Success: Returning %d projects\n", len(pbProjects))

	return connect.NewResponse(&enginev1.ListProjectsResponse{
		Projects: pbProjects,
	}), nil
}

func (s *EngineServer) UpdateProject(
	ctx context.Context,
	req *connect.Request[enginev1.UpdateProjectRequest],
) (*connect.Response[enginev1.UpdateProjectResponse], error) {
	fmt.Printf("[API] UpdateProject Request Received: ID=%s\n", req.Msg.Project.Id)

	// Fetch existing project to ensure it exists
	existingProject, err := s.svc.GetProject(req.Msg.Project.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Update fields
	existingProject.Name = req.Msg.Project.Name
	existingProject.Description = req.Msg.Project.Description
	existingProject.ApiKey = req.Msg.Project.ApiKey
	// Path and Framework usually shouldn't change easily, but we can allow it if needed.
	// For now, let's stick to metadata updates.

	projectDTO := dtos2.Project{
		ID:          existingProject.ID,
		Name:        existingProject.Name,
		Path:        existingProject.Path,
		Framework:   existingProject.Framework,
		Description: existingProject.Description,
		ApiKey:      existingProject.ApiKey,
	}

	if err := s.svc.UpdateProject(projectDTO); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&enginev1.UpdateProjectResponse{
		Project: req.Msg.Project,
	}), nil
}

func (s *EngineServer) DeleteProject(
	ctx context.Context,
	req *connect.Request[enginev1.DeleteProjectRequest],
) (*connect.Response[enginev1.DeleteProjectResponse], error) {
	fmt.Printf("[API] DeleteProject Request Received: ID=%s\n", req.Msg.Id)

	if err := s.svc.DeleteProject(req.Msg.Id); err != nil {
		fmt.Printf("[API] DeleteProject Failed: %v\n", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	fmt.Println("[API] DeleteProject Success")

	return connect.NewResponse(&enginev1.DeleteProjectResponse{
		Success: true,
	}), nil
}

func (s *EngineServer) SyncProject(
	ctx context.Context,
	req *connect.Request[enginev1.SyncProjectRequest],
) (*connect.Response[enginev1.SyncProjectResponse], error) {
	fmt.Printf("[API] SyncProject Request Received: ID=%s\n", req.Msg.Id)

	project, err := s.svc.SyncProject(req.Msg.Id)
	if err != nil {
		fmt.Printf("[API] SyncProject Failed: %v\n", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var pbPages []*enginev1.Page
	for _, page := range project.Pages {
		pbPages = append(pbPages, &enginev1.Page{
			Id:         page.ID,
			Name:       page.Name,
			Route:      page.Route,
			Content:    page.Content,
			RawContent: page.RawContent,
		})
	}

	return connect.NewResponse(&enginev1.SyncProjectResponse{
		Project: &enginev1.Project{
			Id:          project.ID,
			Name:        project.Name,
			Path:        project.Path,
			Settings:    &enginev1.ProjectSettings{Framework: project.Framework},
			Description: project.Description,
			GlobalCss:   project.GlobalCSS,
			Pages:       pbPages,
		},
	}), nil
}

// Page Handlers

func (s *EngineServer) CreatePage(
	ctx context.Context,
	req *connect.Request[enginev1.CreatePageRequest],
) (*connect.Response[enginev1.CreatePageResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("not implemented"))
}

func (s *EngineServer) GetPage(
	ctx context.Context,
	req *connect.Request[enginev1.GetPageRequest],
) (*connect.Response[enginev1.GetPageResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("not implemented"))
}

func (s *EngineServer) SavePage(
	ctx context.Context,
	req *connect.Request[enginev1.SavePageRequest],
) (*connect.Response[enginev1.SavePageResponse], error) {
	fmt.Printf("[API] SavePage Request Received: ProjectID=%s, PageID=%s\n", req.Msg.ProjectId, req.Msg.PageId)

	if err := s.svc.SavePage(req.Msg.ProjectId, req.Msg.PageId, req.Msg.Content); err != nil {
		fmt.Printf("[API] SavePage Failed: %v\n", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&enginev1.SavePageResponse{
		Success: true,
		Message: "Page saved successfully",
	}), nil
}

// Build Handlers

func (s *EngineServer) BuildProject(
	ctx context.Context,
	req *connect.Request[enginev1.BuildProjectRequest],
) (*connect.Response[enginev1.BuildProjectResponse], error) {
	fmt.Printf("[API] BuildProject Request Received: ProjectID=%s\n", req.Msg.ProjectId)

	output, err := s.svc.BuildProject(req.Msg.ProjectId)
	if err != nil {
		fmt.Printf("[API] BuildProject Failed: %v\n", err)
		// Return success=false but still return the output for debugging
		return connect.NewResponse(&enginev1.BuildProjectResponse{
			Success: false,
			Output:  err.Error(),
		}), nil
	}

	return connect.NewResponse(&enginev1.BuildProjectResponse{
		Success: true,
		Output:  output,
	}), nil
}

func (s *EngineServer) RunProject(
	ctx context.Context,
	req *connect.Request[enginev1.RunProjectRequest],
) (*connect.Response[enginev1.RunProjectResponse], error) {
	fmt.Printf("[API] RunProject Request Received: ProjectID=%s\n", req.Msg.ProjectId)

	port, err := s.svc.RunProject(req.Msg.ProjectId)
	if err != nil {
		fmt.Printf("[API] RunProject Failed: %v\n", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&enginev1.RunProjectResponse{
		Success: true,
		Message: fmt.Sprintf("Project running on port %d", port),
		Port:    int32(port),
	}), nil
}

// Plugin & CMS Handlers

func (s *EngineServer) InstallPlugin(
	ctx context.Context,
	req *connect.Request[enginev1.InstallPluginRequest],
) (*connect.Response[enginev1.InstallPluginResponse], error) {
	err := s.svc.InstallPlugin(req.Msg.ProjectId, req.Msg.PluginName, req.Msg.Version)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&enginev1.InstallPluginResponse{
		Success: true,
		Message: fmt.Sprintf("Plugin %s installed successfully", req.Msg.PluginName),
	}), nil
}

func (s *EngineServer) ConfigureCMS(
	ctx context.Context,
	req *connect.Request[enginev1.ConfigureCMSRequest],
) (*connect.Response[enginev1.ConfigureCMSResponse], error) {
	err := s.svc.ConfigureCMS(req.Msg.ProjectId, req.Msg.CmsType, req.Msg.ConfigJson)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&enginev1.ConfigureCMSResponse{
		Success: true,
		Message: fmt.Sprintf("CMS %s configured successfully", req.Msg.CmsType),
	}), nil
}

// Swarm Handlers

func (s *EngineServer) RunSwarm(
	ctx context.Context,
	req *connect.Request[enginev1.RunSwarmRequest],
	stream *connect.ServerStream[enginev1.SwarmEvent],
) error {
	fmt.Printf("[API] RunSwarm Request Received: ProjectID=%s\n", req.Msg.ProjectId)

	events, err := s.svc.RunSwarm(req.Msg.ProjectId, req.Msg.Prompt)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	for event := range events {
		var pbEvent *enginev1.SwarmEvent
		switch event.Type {
		case dtos2.SwarmEventPlan:
			var steps []*enginev1.SwarmStep
			for _, step := range event.Plan.Steps {
				steps = append(steps, &enginev1.SwarmStep{
					Id:          step.ID,
					Description: step.Description,
					Status:      step.Status,
				})
			}
			pbEvent = &enginev1.SwarmEvent{
				Event: &enginev1.SwarmEvent_Plan{
					Plan: &enginev1.SwarmPlan{Steps: steps},
				},
			}
		case dtos2.SwarmEventLog:
			pbEvent = &enginev1.SwarmEvent{
				Event: &enginev1.SwarmEvent_Log{
					Log: &enginev1.SwarmLog{
						StepId:  event.Log.StepID,
						Message: event.Log.Message,
						Level:   event.Log.Level,
					},
				},
			}
		case dtos2.SwarmEventResult:
			pbEvent = &enginev1.SwarmEvent{
				Event: &enginev1.SwarmEvent_Result{
					Result: &enginev1.SwarmResult{
						Success: event.Result.Success,
						Message: event.Result.Message,
					},
				},
			}
		}

		if pbEvent != nil {
			if err := stream.Send(pbEvent); err != nil {
				return err
			}
		}
	}

	return nil
}

// File Watcher Handlers

func (s *EngineServer) SubscribeToFileEvents(
	ctx context.Context,
	req *connect.Request[enginev1.SubscribeToFileEventsRequest],
	stream *connect.ServerStream[enginev1.FileEvent],
) error {
	fmt.Printf("[API] SubscribeToFileEvents Request Received: ProjectID=%s\n", req.Msg.ProjectId)

	events, err := s.svc.SubscribeToFileEvents(req.Msg.ProjectId)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-events:
			if !ok {
				return nil
			}
			// Map fsnotify event to proto
			// fsnotify.Op is a bitmask, but we usually get single events
			eventType := "UNKNOWN"
			if event.Op&fsnotify.Create == fsnotify.Create {
				eventType = "CREATE"
			} else if event.Op&fsnotify.Write == fsnotify.Write {
				eventType = "WRITE"
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				eventType = "REMOVE"
			} else if event.Op&fsnotify.Rename == fsnotify.Rename {
				eventType = "RENAME"
			} else if event.Op&fsnotify.Chmod == fsnotify.Chmod {
				eventType = "CHMOD"
			}

			if err := stream.Send(&enginev1.FileEvent{
				Path: event.Name,
				Type: eventType,
			}); err != nil {
				return err
			}
		}
	}
}
