package routers

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/paakofiaidoo/juki/engine/pkg/controllers"
	"net/http"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

type ApiRouter interface {
	HealthRoutes()
	ApiRoutes(controller controllers.Controller)
	Engine() interface{}
}

type router struct {
	router *echo.Echo
}

func (engine *router) HealthRoutes() {
	v1 := engine.router.Group("/healthz")
	v1.GET("/ready", func(c echo.Context) error {
		return c.String(http.StatusOK, "Ready!")
	})

	v1.GET("/live", func(ctx echo.Context) error {
		return ctx.String(http.StatusOK, "Ready!")
	})
}

func (engine *router) Engine() interface{} {
	return engine.router
}

func (engine *router) ApiRoutes(controller controllers.Controller) {
	api := humaecho.New(engine.router, huma.DefaultConfig("My API", "1.0.0"))

	huma.Register(api, huma.Operation{
		OperationID:   "pages-create",
		Method:        http.MethodPost,
		Path:          "/pages",
		Summary:       "Create new Page",
		DefaultStatus: http.StatusCreated,
	}, controller.CreatePage)

	huma.Register(api, huma.Operation{
		OperationID:   "app-create",
		Method:        http.MethodPost,
		Path:          "/apps",
		Summary:       "Create App",
		DefaultStatus: http.StatusCreated,
	}, controller.CreatePage)

}

func New(app *echo.Echo) ApiRouter {
	app.Use(echoMiddleware.Logger())

	return &router{
		router: app,
	}
}