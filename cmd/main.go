package main

import (
	"fmt"
	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/labstack/echo/v4"
	"github.com/paakofiaidoo/juki/engine/data/repositories"
	"github.com/paakofiaidoo/juki/engine/pkg/config"
	"github.com/paakofiaidoo/juki/engine/pkg/controllers"
	"github.com/paakofiaidoo/juki/engine/pkg/database"
	"github.com/paakofiaidoo/juki/engine/pkg/routers"
	"github.com/paakofiaidoo/juki/engine/pkg/scripts"
	"github.com/paakofiaidoo/juki/engine/pkg/services"
	"net/http"

	"gorm.io/gorm"
)

type Options struct {
	Port int `help:"Port to listen on" short:"p" default:"4220"`
}

/* ============================================
*			Variable Initializers
* ============================================*/
var (
	app    = echo.New()
	cfg    = config.New()
	db     = database.New()
	router = routers.New(app)
)

/* ============================================
*			Setup Initializers
* ============================================*/
func init() {
	cfg.LoadEnv()
	db.SetConfig(cfg.DBConfig())
	db.Connect()

}

/* ============================================
*			App Entrypoint
* ============================================*/
func main() {
	/* ============================================
	*			App Configurations
	* ============================================*/
	dbHandler := db.Engine().(*gorm.DB)

	/* ============================================
	 *			Middlewares
	 * ============================================*/

	repository := repositories.NewRepository(dbHandler)
	/* ============================================
	 *			Repositories
	 * ============================================*/
	script := scripts.NewScript()

	service := services.NewService(repository, script)

	/* ============================================
	*			Controllers
	* ============================================*/
	controller := controllers.NewController(service, script)
	/* ============================================
	*			Router
	* ============================================*/

	/* ============================================
	*			App Entrypoint
	* ============================================*/

	cli := humacli.New(func(hooks humacli.Hooks, options *Options) {

		router.ApiRoutes(controller)

		hooks.OnStart(func() {
			fmt.Printf("Starting server on port   http://localhost:%d...\n", options.Port)
			err := http.ListenAndServe(fmt.Sprintf(":%d", options.Port), app)
			if err != nil {
				return
			}
		})
	})

	// Run the CLI. When passed no commands, it starts the server.
	cli.Run()
}