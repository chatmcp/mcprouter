package router

import (
	"github.com/chatmcp/mcprouter/handler/api/beta"
	v1 "github.com/chatmcp/mcprouter/handler/api/v1"
	"github.com/chatmcp/mcprouter/service/api"

	"github.com/labstack/echo/v4"
)

// APIRoute will create the routes for the http server
func APIRoute(e *echo.Echo) {
	apiv1beta := e.Group("/beta")
	apiv1beta.Use(api.CreateAPIBetaMiddleware())
	apiv1beta.POST("/add-server", beta.AddServer)
	apiv1beta.POST("/update-server", beta.UpdateServer)
	apiv1beta.POST("/get-servers", beta.GetServers)
	apiv1beta.POST("/get-server", beta.GetServer)
	apiv1beta.POST("/get-user", beta.GetUser)
	apiv1beta.POST("/save-user", beta.SaveUser)

	apiv1 := e.Group("/v1")
	apiv1.Use(api.CreateAPIV1Middleware())
	apiv1.POST("/list-servers", v1.ListServers)
	apiv1.POST("/get-server", v1.GetServer)
	apiv1.POST("/list-tools", v1.ListTools)
	apiv1.POST("/call-tool", v1.CallTool)
}
