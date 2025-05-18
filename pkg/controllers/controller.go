package controllers

import (
	"context"
	"github.com/paakofiaidoo/juki/engine/data/domains"
	"github.com/paakofiaidoo/juki/engine/data/dtos"
	"github.com/paakofiaidoo/juki/engine/pkg/scripts"
	"github.com/paakofiaidoo/juki/engine/pkg/services"
)

type Controller interface {
	CreatePage(ctx context.Context, input *dtos.PageDto) (*domains.PageDomain, error)
}

type controller struct {
	service services.Service
}

func NewController(service services.Service, script scripts.Scripts) Controller {
	return &controller{
		service: service,
	}
}