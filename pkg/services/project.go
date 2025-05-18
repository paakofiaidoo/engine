package services

import (
	"errors"
	"github.com/paakofiaidoo/juki/engine/data/dtos"
	"log"
)

func (s *service) CreateProject(project dtos.Project) error {

	log.Println("Create Project ....")

	switch project.Framework {
	case "NextJS":
		s.scripts.CreateAppNextApp(project)
	default:
		return errors.New("not supported framework")

	}
	return nil

}

func (s *service) UpdateProject(project dtos.Project) error {

	return nil
}

func (s *service) DeleteProject(project dtos.Project) error {
	return nil
}
func (s *service) GetProjectById(id int) error {
	project, err := s.repository.GetProject(id)

}