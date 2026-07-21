package api

import (
	"context"
	"fmt"
	"juki-engine/pkg/data/models"
	"juki-engine/pkg/data/repositories"
	enginev1 "juki-engine/pkg/gen/juki/engine/v1"

	"connectrpc.com/connect"
)

type ComponentServer struct {
	repo repositories.Repository
}

func NewComponentServer(repo repositories.Repository) *ComponentServer {
	return &ComponentServer{repo: repo}
}

func (s *ComponentServer) CreateComponent(
	ctx context.Context,
	req *connect.Request[enginev1.CreateComponentRequest],
) (*connect.Response[enginev1.CreateComponentResponse], error) {
	fmt.Printf("[API] CreateComponent: ProjectID=%s, Name=%s\n", req.Msg.ProjectId, req.Msg.Name)

	component := &models.UserComponent{
		ProjectID: req.Msg.ProjectId,
		Name:      req.Msg.Name,
		Content:   req.Msg.Content,
	}

	if err := s.repo.CreateComponent(component); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&enginev1.CreateComponentResponse{
		Component: &enginev1.UserComponent{
			Id:        component.ID,
			ProjectId: component.ProjectID,
			Name:      component.Name,
			Content:   component.Content,
		},
	}), nil
}

func (s *ComponentServer) UpdateComponent(
	ctx context.Context,
	req *connect.Request[enginev1.UpdateComponentRequest],
) (*connect.Response[enginev1.UpdateComponentResponse], error) {
	fmt.Printf("[API] UpdateComponent: ID=%s\n", req.Msg.Id)

	component, err := s.repo.GetComponent(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if req.Msg.Name != "" {
		component.Name = req.Msg.Name
	}
	if req.Msg.Content != "" {
		component.Content = req.Msg.Content
	}

	if err := s.repo.UpdateComponent(component); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&enginev1.UpdateComponentResponse{
		Component: &enginev1.UserComponent{
			Id:        component.ID,
			ProjectId: component.ProjectID,
			Name:      component.Name,
			Content:   component.Content,
		},
	}), nil
}

func (s *ComponentServer) DeleteComponent(
	ctx context.Context,
	req *connect.Request[enginev1.DeleteComponentRequest],
) (*connect.Response[enginev1.DeleteComponentResponse], error) {
	fmt.Printf("[API] DeleteComponent: ID=%s\n", req.Msg.Id)

	if err := s.repo.DeleteComponent(req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&enginev1.DeleteComponentResponse{
		Success: true,
	}), nil
}

func (s *ComponentServer) ListComponents(
	ctx context.Context,
	req *connect.Request[enginev1.ListComponentsRequest],
) (*connect.Response[enginev1.ListComponentsResponse], error) {
	fmt.Printf("[API] ListComponents: ProjectID=%s\n", req.Msg.ProjectId)

	components, err := s.repo.ListComponents(req.Msg.ProjectId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var pbComponents []*enginev1.UserComponent
	for _, c := range components {
		pbComponents = append(pbComponents, &enginev1.UserComponent{
			Id:        c.ID,
			ProjectId: c.ProjectID,
			Name:      c.Name,
			Content:   c.Content,
		})
	}

	return connect.NewResponse(&enginev1.ListComponentsResponse{
		Components: pbComponents,
	}), nil
}
