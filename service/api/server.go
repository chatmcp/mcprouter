package api

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// APIServer as the proxy server for API request
type APIServer struct {
	server *echo.Echo // http server built with echo
}

// NewAPIServer will create API server
func NewAPIServer() *APIServer {
	return &APIServer{
		server: echo.New(),
	}
}

// Route will create the routes for http server
func (s *APIServer) Route(route func(e *echo.Echo)) {
	s.server.Validator = NewValidator()
	s.server.Use(middleware.Logger())
	s.server.Use(CreateAPIMiddleware())

	route(s.server)
}

// Start will start the http server
func (s *APIServer) Start(port int) {
	s.server.Logger.Fatal(s.server.Start(fmt.Sprintf(":%d", port)))
}
