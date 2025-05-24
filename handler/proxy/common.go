package proxy

import (
	"net/http"
	"time"

	"github.com/chatmcp/mcprouter/service/mcpserver"
	"github.com/chatmcp/mcprouter/service/proxy"
	"github.com/labstack/echo/v4"
)

// Constants for better maintainability
const (
	AcceptJSON         = "application/json"
	AcceptEventStream  = "text/event-stream"
	HeaderMcpSessionID = "Mcp-Session-Id"
	HeaderXRequestID   = "X-Request-ID"
	HeaderXRequestFrom = "X-Request-From"

	MethodInitialize = "initialize"
	MethodToolsCall  = "tools/call"
)

// HTTP response messages
const (
	ErrFailedSSEContext    = "Failed to get SSE context"
	ErrKeyRequired         = "Key is required"
	ErrInvalidServerConfig = "Invalid server config"
	ErrInvalidSessionID    = "Invalid session ID"
	ErrMethodNotAllowed    = "Method Not Allowed"
	ErrGETRequiresSSE      = "GET requests require text/event-stream Accept header"
)

// validateKeyAndConfig validates key parameter and retrieves server configuration
func validateKeyAndConfig(c echo.Context) (string, *mcpserver.ServerConfig, error) {
	// Validate key parameter
	key := c.Param("key")
	if key == "" {
		return "", nil, c.String(http.StatusBadRequest, "Key is required")
	}

	// Get server configuration
	serverConfig := mcpserver.GetServerConfig(key)
	if serverConfig == nil {
		return "", nil, c.String(http.StatusBadRequest, "Invalid server config")
	}

	return key, serverConfig, nil
}

// validateSSEContext validates and returns SSE context
func validateSSEContext(c echo.Context) (*proxy.SSEContext, error) {
	ctx := proxy.GetSSEContext(c)
	if ctx == nil {
		return nil, c.String(http.StatusInternalServerError, "Failed to get context")
	}
	return ctx, nil
}

// createProxyInfo creates basic ProxyInfo structure with server config
func createProxyInfo(key string, serverConfig *mcpserver.ServerConfig) *proxy.ProxyInfo {
	return &proxy.ProxyInfo{
		ServerKey:          key,
		ConnectionTime:     time.Now(),
		ServerUUID:         serverConfig.ServerUUID,
		ServerConfigName:   serverConfig.ServerName,
		ServerShareProcess: serverConfig.ShareProcess,
		ServerType:         serverConfig.ServerType,
		ServerURL:          serverConfig.ServerURL,
		ServerCommand:      serverConfig.Command,
		ServerCommandHash:  serverConfig.CommandHash,
	}
}
