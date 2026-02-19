package main

import (
	"fmt"
	"net/http"
	"time"

	"juki-engine/pkg/bridge"
	"juki-engine/pkg/config"
	"juki-engine/pkg/data/repositories"
	"juki-engine/pkg/database"
	enginev1connect "juki-engine/pkg/gen/juki/engine/v1/v1connect"
	jukimiddleware "juki-engine/pkg/middleware"
	"juki-engine/pkg/scripts"
	api "juki-engine/pkg/servers"
	"juki-engine/pkg/services"
	"juki-engine/pkg/system"
	"juki-engine/pkg/watcher"

	"connectrpc.com/connect"
	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Options struct {
	Port int `help:"Port to listen on" short:"p" default:"4220"`
}

/* ============================================
*			Variable Initializers
* ============================================*/
var (
	app = echo.New()
	cfg = config.New()
	db  = database.New()
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
	// Ensure system dependencies (Mac only for now)
	if err := system.EnsureDependencies(); err != nil {
		fmt.Printf("Warning: Failed to check/install dependencies: %v\n", err)
	}

	/* ============================================
	*			App Configurations
	* ============================================*/
	dbHandler := db.Engine()

	/* ============================================
	 *			Middlewares
	 * ============================================*/
	app.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, "Connect-Protocol-Version"},
	}))

	repository := repositories.NewRepository(dbHandler)
	/* ============================================
	 *			Repositories
	 * ============================================*/
	script := scripts.NewScript()

	// Initialize Bridge
	bridge, err := bridge.NewBridge("worker.ts")
	if err != nil {
		fmt.Printf("Warning: Failed to start Node worker: %v\n", err)
	}
	// defer bridge.Close() // TODO: Handle graceful shutdown

	// Initialize Watcher
	watcher, err := watcher.NewWatcher(100 * time.Millisecond)
	if err != nil {
		fmt.Printf("Warning: Failed to start file watcher: %v\n", err)
	} else {
		watcher.Start()
		defer watcher.Close()
	}

	service := services.NewService(repository, script, bridge, watcher)

	/* ============================================
	*			Controllers
	* ============================================*/
	/* ============================================
	*			Router
	* ============================================*/

	/* ============================================
	*			App Entrypoint
	* ============================================*/

	// ConnectRPC Setup
	activityLogger := jukimiddleware.NewActivityLogger(repository)
	interceptorOpt := connect.WithInterceptors(activityLogger)

	// Engine Service
	engineServer := api.NewEngineServer(service)
	enginePath, engineHandler := enginev1connect.NewEngineServiceHandler(engineServer, interceptorOpt)
	app.Any(enginePath+"*", echo.WrapHandler(engineHandler))

	// Terminal Service
	terminalService := services.NewTerminalService(repository)
	terminalServer := api.NewTerminalServer(terminalService)
	terminalPath, terminalHandler := enginev1connect.NewTerminalServiceHandler(terminalServer, interceptorOpt)
	app.Any(terminalPath+"*", echo.WrapHandler(terminalHandler))

	// Activity Service
	activityServer := api.NewActivityServer()
	activityPath, activityHandler := enginev1connect.NewActivityServiceHandler(activityServer)
	app.Any(activityPath+"*", echo.WrapHandler(activityHandler))

	cli := humacli.New(func(hooks humacli.Hooks, options *Options) {

		//router.ApiRoutes(controller)

		hooks.OnStart(func() {
			fmt.Printf("Starting server on port   http://localhost:%d...\n", options.Port)
			// Enable HTTP/2 support if possible, though Echo standard listener might handle it if TLS is enabled or h2c is used.
			// For now, we use the standard listener.
			err := http.ListenAndServe(fmt.Sprintf(":%d", options.Port), app)
			if err != nil {
				fmt.Printf("Error starting server: %v\n", err)
				return
			}
		})
	})

	// Run the CLI. When passed no commands, it starts the server.
	cli.Run()
}
