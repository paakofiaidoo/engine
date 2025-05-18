package scripts

import (
	"github.com/paakofiaidoo/juki/engine/data/dtos"
)

type Scripts interface {
	CreateAppNextApp(project dtos.Project)
}

type scripts struct {
}

func (s scripts) CreateAppNextApp(app dtos.Project) {
	// theis is to run the script to create a nextjs app
	// first we must check if node is installed
	// if version is less than 18 through and error
	//eles run npx create-next-app@latest
}

func NewScript() Scripts {
	return &scripts{}
}