package services

import (
	"github.com/paakofiaidoo/juki/engine/data/repositories"
	"github.com/paakofiaidoo/juki/engine/pkg/scripts"
)

/* ============================================
*			Services
* ============================================*/

type Service interface {
}

type service struct {
	repository repositories.Repository
	scripts    scripts.Scripts
}

/* ============================================
*			Service Constructors
* ============================================*/

func NewService(repository repositories.Repository, script scripts.Scripts) Service {
	return &service{
		repository: repository,
		scripts:    script,
	}
}