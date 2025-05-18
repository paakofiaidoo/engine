package domains

import (
	"encoding/json"
	"github.com/jinzhu/copier"
	"github.com/paakofiaidoo/juki/engine/data/models"
	"log"
	"time"
)

/* ============================================
*			Domains
* ============================================*/

type PageDomain struct {
	ID          string
	Name        string
	Route       string
	CustomTheme bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ProjectDomain struct {
	Id          int
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

/* ============================================
*			Struct Methods
* ============================================*/

func NewPageDomain(page *models.Page) *PageDomain {
	pageDomain := new(PageDomain)
	if err := copier.Copy(pageDomain, page); err != nil {
		log.Println("failed to copy model to page:", err)
	}
	return pageDomain
}

func (d PageDomain) String() string {
	jsonBytes, _ := json.Marshal(d)
	return string(jsonBytes)
}