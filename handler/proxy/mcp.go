package proxy

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/chatmcp/mcprouter/model"
	"github.com/chatmcp/mcprouter/service/jsonrpc"
	"github.com/chatmcp/mcprouter/service/mcpclient"
	"github.com/chatmcp/mcprouter/service/mcpserver"
	"github.com/chatmcp/mcprouter/service/proxy"
	"github.com/chatmcp/mcprouter/util"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

// MCP is the main handler for the MCP endpoint
func MCP(c echo.Context) error {
	ctx, err := validateSSEContext(c)
	if err != nil {
		return err
	}

	switch c.Request().Method {
	case http.MethodOptions:
		return handleCORS(c)
	case http.MethodGet:
		return handleSSE(c, ctx)
	case http.MethodDelete:
		return cleanupSession(c, ctx)
	case http.MethodPost:
		return processRequest(c, ctx)
	default:
		return c.String(http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}

// handleCORS handles CORS preflight requests
func handleCORS(c echo.Context) error {
	response := c.Response()
	response.Header().Set("Access-Control-Allow-Origin", "*")
	response.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization, Mcp-Session-Id")
	response.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	response.Header().Set("Access-Control-Max-Age", "86400")
	return c.NoContent(http.StatusOK)
}

// processRequest handles POST requests for MCP communication
func processRequest(c echo.Context, ctx *proxy.SSEContext) error {
	// Parse and validate request
	key, serverConfig, request, err := parseRequest(c, ctx)
	if err != nil {
		return err
	}

	// Handle notification requests early
	if request.Result != nil || request.Error != nil {
		return ctx.JSONRPCAcceptResponse(nil)
	}

	// Setup session and proxy info
	proxyInfo, sessionID, err := setupSession(c, ctx, key, serverConfig, request)
	if err != nil {
		return err
	}

	// Forward request to MCP server
	response, err := forwardRequest(ctx, key, serverConfig, request)
	if err != nil {
		return err
	}

	// Process initialize response if needed
	if err := processInitResponse(request, response, proxyInfo, sessionID); err != nil {
		return ctx.JSONRPCError(jsonrpc.ErrorParseError, request.ID)
	}

	// Send final response
	return sendResponse(c, ctx, proxyInfo, request, response)
}

// parseRequest validates the request and parses JSON-RPC
func parseRequest(c echo.Context, ctx *proxy.SSEContext) (string, *mcpserver.ServerConfig, *jsonrpc.Request, error) {
	// Use common validation function
	key, serverConfig, err := validateKeyAndConfig(c)
	if err != nil {
		return "", nil, nil, err
	}

	// Parse JSON-RPC request
	request, err := ctx.GetJSONRPCRequest()
	if err != nil {
		return "", nil, nil, ctx.JSONRPCError(jsonrpc.ErrorParseError, nil)
	}

	return key, serverConfig, request, nil
}

// setupSession creates proxy info and manages session
func setupSession(c echo.Context, ctx *proxy.SSEContext, key string, serverConfig *mcpserver.ServerConfig, request *jsonrpc.Request) (*proxy.ProxyInfo, string, error) {
	// Create base proxy info using common function
	proxyInfo := createProxyInfo(key, serverConfig)

	// Add MCP-specific fields
	header := c.Request().Header
	proxyInfo.SessionID = header.Get(HeaderMcpSessionID)
	proxyInfo.RequestID = header.Get(HeaderXRequestID)
	proxyInfo.RequestFrom = header.Get(HeaderXRequestFrom)
	proxyInfo.JSONRPCVersion = request.JSONRPC
	proxyInfo.RequestMethod = request.Method
	proxyInfo.RequestTime = time.Now()
	proxyInfo.RequestParams = request.Params

	if request.ID != nil {
		proxyInfo.RequestID = request.ID
	}

	var sessionID string
	var err error

	if request.Method == MethodInitialize {
		sessionID, err = createSession(c, ctx, proxyInfo, request)
	} else {
		sessionID, err = loadSession(c, proxyInfo)
	}

	return proxyInfo, sessionID, err
}

// createSession creates a new session for initialize requests
func createSession(c echo.Context, ctx *proxy.SSEContext, proxyInfo *proxy.ProxyInfo, request *jsonrpc.Request) (string, error) {
	// Parse initialize parameters
	paramsBytes, err := json.Marshal(request.Params)
	if err != nil {
		return "", ctx.JSONRPCError(jsonrpc.ErrorParseError, nil)
	}

	var params jsonrpc.InitializeParams
	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		return "", ctx.JSONRPCError(jsonrpc.ErrorParseError, nil)
	}

	// Generate session ID using common function
	sessionID := proxyInfo.GetSessionID()
	proxyInfo.ConnectionTime = time.Now()
	proxyInfo.ClientName = params.ClientInfo.Name
	proxyInfo.ClientVersion = params.ClientInfo.Version
	proxyInfo.ProtocolVersion = params.ProtocolVersion
	proxyInfo.SessionID = sessionID

	// Store proxy info
	if err := proxy.StoreProxyInfo(sessionID, proxyInfo); err != nil {
		log.Printf("Failed to store proxy info: %v", err)
	}

	// Set session ID in response header
	c.Response().Header().Set(HeaderMcpSessionID, sessionID)

	log.Printf("Created new session: %s for client: %s", sessionID, proxyInfo.ClientName)
	return sessionID, nil
}

// loadSession validates and retrieves existing session data
func loadSession(c echo.Context, proxyInfo *proxy.ProxyInfo) (string, error) {
	sessionID := proxyInfo.SessionID
	if sessionID == "" {
		return "", c.String(http.StatusBadRequest, ErrInvalidSessionID)
	}

	// Try to get existing proxy info and merge relevant data
	if existingInfo, err := proxy.GetProxyInfo(sessionID); err == nil && existingInfo != nil && existingInfo.SessionID == sessionID {
		proxyInfo.ClientName = existingInfo.ClientName
		proxyInfo.ClientVersion = existingInfo.ClientVersion
		proxyInfo.ProtocolVersion = existingInfo.ProtocolVersion
		proxyInfo.ConnectionTime = existingInfo.ConnectionTime
		proxyInfo.ServerName = existingInfo.ServerName
		proxyInfo.ServerVersion = existingInfo.ServerVersion
	}

	log.Printf("Using existing session: %s", sessionID)
	return sessionID, nil
}

// forwardRequest handles MCP client operations and request forwarding
func forwardRequest(ctx *proxy.SSEContext, key string, serverConfig *mcpserver.ServerConfig, request *jsonrpc.Request) (*jsonrpc.Response, error) {
	// Get existing client or create new one
	client := ctx.GetClient(key)
	if client == nil {
		newClient, err := mcpclient.NewClient(serverConfig)
		if err != nil {
			log.Printf("Failed to connect to MCP server: %v", err)
			return nil, ctx.JSONRPCError(jsonrpc.ErrorProxyError, request.ID)
		}

		if err := newClient.Error(); err != nil {
			log.Printf("MCP server run failed: %v", err)
			return nil, ctx.JSONRPCError(jsonrpc.ErrorProxyError, request.ID)
		}

		// Set up notification handler
		newClient.OnNotification(func(message []byte) {
			log.Printf("Received notification: %s", message)
		})

		ctx.StoreClient(key, newClient)
		client = newClient
	}

	// Forward message to MCP server
	response, err := client.ForwardMessage(request)
	if err != nil {
		log.Printf("Failed to forward message: %v", err)
		client.Close()
		ctx.DeleteClient(key)
		return nil, ctx.JSONRPCError(jsonrpc.ErrorProxyError, request.ID)
	}

	return response, nil
}

// processInitResponse processes initialize method responses
func processInitResponse(request *jsonrpc.Request, response *jsonrpc.Response, proxyInfo *proxy.ProxyInfo, sessionID string) error {
	if request.Method != MethodInitialize || response == nil || response.Result == nil {
		return nil
	}

	// Extract initialize result
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return err
	}

	var result jsonrpc.InitializeResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		log.Printf("Failed to unmarshal initialize result: %v", err)
		return err
	}

	// Update proxy info with server information
	proxyInfo.ServerName = result.ServerInfo.Name
	proxyInfo.ServerVersion = result.ServerInfo.Version

	// Store updated proxy info
	if err := proxy.StoreProxyInfo(sessionID, proxyInfo); err != nil {
		log.Printf("Failed to store proxy info with server info: %v", err)
	}

	log.Printf("Updated proxy info with server info: %s, server: %s", sessionID, proxyInfo.ServerName)
	return nil
}

// sendResponse finalizes processing and sends the response
func sendResponse(c echo.Context, ctx *proxy.SSEContext, proxyInfo *proxy.ProxyInfo, request *jsonrpc.Request, response *jsonrpc.Response) error {
	// Update response timing and proxy info
	proxyInfo.ResponseResult = response
	proxyInfo.ResponseTime = time.Now()
	if !proxyInfo.RequestTime.IsZero() {
		costTime := proxyInfo.ResponseTime.Sub(proxyInfo.RequestTime)
		proxyInfo.CostTime = costTime.Milliseconds()
	}

	// Save logs if enabled
	if viper.GetBool("app.save_log") && proxyInfo.RequestMethod == MethodToolsCall {
		if err := model.CreateServerLog(proxyInfo.ToServerLog()); err != nil {
			log.Printf("Failed to save server log: %v", err)
		} else {
			log.Printf("Saved server log successfully: %v", proxyInfo.RequestID)
		}
	}

	// Log proxy info for debugging
	if proxyInfoBytes, err := json.Marshal(proxyInfo); err == nil {
		log.Printf("Proxy info: %s", string(proxyInfoBytes))
	}

	// Handle notification response
	if response == nil {
		return ctx.JSONRPCAcceptResponse(response)
	}

	// Determine response format and send
	return writeResponse(c, ctx, response)
}

// writeResponse determines and sends the appropriate response format
func writeResponse(c echo.Context, ctx *proxy.SSEContext, response *jsonrpc.Response) error {
	accept := c.Request().Header.Get("Accept")
	acceptValues := strings.Split(accept, ",")

	// Check if event-stream should be used
	useEventStream := false
	for i, val := range acceptValues {
		trimmed := strings.TrimSpace(val)
		if trimmed == AcceptEventStream {
			useEventStream = true
			break
		} else if trimmed == AcceptJSON && i == 0 {
			useEventStream = false
			break
		}
	}

	if useEventStream {
		// Send streaming SSE response with fallback
		if err := ctx.JSONRPCStreamResponse(response); err != nil {
			log.Printf("Failed to send SSE response: %v", err)
			return ctx.JSONRPCResponse(response)
		}
		return nil
	}

	// Send regular JSON response
	return ctx.JSONRPCResponse(response)
}

// handleSSE handles GET requests for establishing persistent SSE connections
func handleSSE(c echo.Context, ctx *proxy.SSEContext) error {
	key := c.Param("key")

	// Validate that this is an event-stream request
	accept := c.Request().Header.Get("Accept")
	if !strings.Contains(accept, AcceptEventStream) {
		return c.String(http.StatusBadRequest, ErrGETRequiresSSE)
	}

	// Validate server configuration
	if mcpserver.GetServerConfig(key) == nil {
		return c.String(http.StatusBadRequest, ErrInvalidServerConfig)
	}

	// Start SSE stream
	writer, err := ctx.JSONRPCStreamStart()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to start SSE stream")
	}

	// Maintain connection
	writer.SendEventData("connection", "ready")
	<-c.Request().Context().Done()
	writer.SendEventData("connection", "closed")

	return nil
}

// cleanupSession handles DELETE requests for session cleanup
func cleanupSession(c echo.Context, ctx *proxy.SSEContext) error {
	key := c.Param("key")

	// Get session ID for cleanup
	sessionID := c.Request().Header.Get(HeaderMcpSessionID)
	if sessionID == "" {
		sessionID = util.MD5(key)
	}

	// Clean up resources
	ctx.DeleteClient(key)
	ctx.DeleteSession(sessionID)

	if err := proxy.DeleteProxyInfo(sessionID); err != nil {
		log.Printf("Failed to delete proxy info for session %s: %v", sessionID, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message":   "Session cleaned up successfully",
		"sessionId": sessionID,
		"key":       key,
	})
}
