package proxy

import (
	"fmt"
	"sync"

	"github.com/labstack/echo/v4"
)

// SSEServer as the proxy server for SSE request
type SSEServer struct {
	server   *echo.Echo // http server built with echo
	sessions *sync.Map  // sessions store
}

// NewSSEServer will create SSE server
func NewSSEServer() *SSEServer {
	return &SSEServer{
		server:   echo.New(),
		sessions: &sync.Map{},
	}
}

// Route will create the routes for http server
func (s *SSEServer) Route(route func(e *echo.Echo)) {
	s.server.Use(createSSEMiddleware(s.sessions))
	route(s.server)
}

// Start will start the http server
func (s *SSEServer) Start(port int) {
	s.server.Logger.Fatal(s.server.Start(fmt.Sprintf(":%d", port)))
}
