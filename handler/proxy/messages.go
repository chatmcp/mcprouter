package proxy

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/chatmcp/mcprouter/service/jsonrpc"
	"github.com/chatmcp/mcprouter/service/mcpclient"
	"github.com/chatmcp/mcprouter/service/proxy"
	"github.com/labstack/echo/v4"
)

// Messages is a handler for the messages endpoint
func Messages(c echo.Context) error {
	// Validate context and parse request
	ctx, session, request, err := parseMessageRequest(c)
	if err != nil {
		return err
	}

	// Setup message session and proxy info
	proxyInfo, sseKey, err := setupMessageSession(ctx, session, request)
	if err != nil {
		return err
	}

	// Process message with MCP client
	response, err := processMessageWithClient(ctx, session, sseKey, request)
	if err != nil {
		return err
	}

	// Handle message response and finalize
	return handleMessageResponse(ctx, session, request, response, proxyInfo)
}

// parseMessageRequest validates context and parses message request
func parseMessageRequest(c echo.Context) (*proxy.SSEContext, *proxy.SSESession, *jsonrpc.Request, error) {
	// Validate SSE context using common function
	ctx, err := validateSSEContext(c)
	if err != nil {
		return nil, nil, nil, err
	}

	// Validate session ID parameter
	sessionID := ctx.QueryParam("sessionid")
	if sessionID == "" {
		return nil, nil, nil, ctx.JSONRPCError(jsonrpc.ErrorInvalidParams, nil)
	}

	// Get session from context
	session := ctx.GetSession(sessionID)
	if session == nil {
		return nil, nil, nil, ctx.JSONRPCError(jsonrpc.ErrorInvalidParams, nil)
	}

	// Parse JSON-RPC request
	request, err := ctx.GetJSONRPCRequest()
	if err != nil {
		return nil, nil, nil, ctx.JSONRPCError(jsonrpc.ErrorParseError, nil)
	}

	return ctx, session, request, nil
}

// setupMessageSession configures session and proxy info for message processing
func setupMessageSession(ctx *proxy.SSEContext, session *proxy.SSESession, request *jsonrpc.Request) (*proxy.ProxyInfo, string, error) {
	proxyInfo := session.ProxyInfo()
	sseKey := session.Key()

	// Update proxy info with request details
	proxyInfo.JSONRPCVersion = request.JSONRPC
	proxyInfo.RequestMethod = request.Method
	proxyInfo.RequestTime = time.Now()
	proxyInfo.RequestParams = request.Params

	if request.ID != nil {
		proxyInfo.RequestID = request.ID
	}

	// Handle initialize method specially
	if request.Method == "initialize" {
		if err := processInitializeParams(ctx, session, proxyInfo, request); err != nil {
			return nil, "", err
		}
	}

	return proxyInfo, sseKey, nil
}

// processInitializeParams handles initialize method parameters
func processInitializeParams(ctx *proxy.SSEContext, session *proxy.SSESession, proxyInfo *proxy.ProxyInfo, request *jsonrpc.Request) error {
	// Parse initialize parameters
	paramsBytes, _ := json.Marshal(request.Params)
	params := &jsonrpc.InitializeParams{}
	if err := json.Unmarshal(paramsBytes, params); err != nil {
		return ctx.JSONRPCError(jsonrpc.ErrorParseError, nil)
	}

	// Update proxy info with client information
	proxyInfo.ClientName = params.ClientInfo.Name
	proxyInfo.ClientVersion = params.ClientInfo.Version
	proxyInfo.ProtocolVersion = params.ProtocolVersion

	// Store updated session
	session.SetProxyInfo(proxyInfo)
	ctx.StoreSession(session.ProxyInfo().SessionID, session)

	return nil
}

// processMessageWithClient handles MCP client operations and message forwarding
func processMessageWithClient(ctx *proxy.SSEContext, session *proxy.SSESession, sseKey string, request *jsonrpc.Request) (*jsonrpc.Response, error) {
	// Get or create MCP client
	client := ctx.GetClient(sseKey)
	if client == nil {
		newClient, err := createMCPClient(ctx, session, sseKey)
		if err != nil {
			return nil, ctx.JSONRPCError(jsonrpc.ErrorProxyError, request.ID)
		}
		client = newClient
	}

	// Forward message to MCP server
	response, err := client.ForwardMessage(request)
	if err != nil {
		fmt.Printf("Forward message failed: %v\n", err)
		session.Close()
		ctx.DeleteClient(sseKey)
		return nil, ctx.JSONRPCError(jsonrpc.ErrorProxyError, request.ID)
	}

	return response, nil
}

// createMCPClient creates and configures a new MCP client
func createMCPClient(ctx *proxy.SSEContext, session *proxy.SSESession, sseKey string) (mcpclient.Client, error) {
	client, err := mcpclient.NewClient(session.ServerConfig())
	if err != nil {
		fmt.Printf("Connect to MCP server failed: %v\n", err)
		return nil, err
	}

	if err := client.Error(); err != nil {
		fmt.Printf("MCP server run failed: %v\n", err)
		return nil, err
	}

	// Set up notification handler
	client.OnNotification(func(message []byte) {
		fmt.Printf("Received notification: %s\n", message)
		session.SendMessage(string(message))
	})

	// Store client in context
	ctx.StoreClient(sseKey, client)

	return client, nil
}

// handleMessageResponse processes response and finalizes message handling
func handleMessageResponse(ctx *proxy.SSEContext, session *proxy.SSESession, request *jsonrpc.Request, response *jsonrpc.Response, proxyInfo *proxy.ProxyInfo) error {
	if response != nil {
		// Handle initialize response specially
		if request.Method == "initialize" && response.Result != nil {
			if err := processInitializeResponse(session, proxyInfo, response); err != nil {
				return ctx.JSONRPCError(jsonrpc.ErrorParseError, request.ID)
			}
		}

		// Send SSE message for non-notification responses
		session.SendMessage(response.String())
	}

	// Update response timing and proxy info
	proxyInfo.ResponseResult = response
	proxyInfo.ResponseTime = time.Now()
	if !proxyInfo.RequestTime.IsZero() {
		costTime := proxyInfo.ResponseTime.Sub(proxyInfo.RequestTime)
		proxyInfo.CostTime = costTime.Milliseconds()
	}

	// Log proxy info for debugging
	if proxyInfoBytes, err := json.Marshal(proxyInfo); err == nil {
		fmt.Printf("Proxy info: %s\n", string(proxyInfoBytes))
	}

	return ctx.JSONRPCResponse(response)
}

// processInitializeResponse handles initialize method response
func processInitializeResponse(session *proxy.SSESession, proxyInfo *proxy.ProxyInfo, response *jsonrpc.Response) error {
	// Parse initialize result
	resultBytes, _ := json.Marshal(response.Result)
	result := &jsonrpc.InitializeResult{}
	if err := json.Unmarshal(resultBytes, result); err != nil {
		fmt.Printf("Unmarshal initialize result failed: %v\n", err)
		return err
	}

	// Update proxy info with server information
	proxyInfo.ServerName = result.ServerInfo.Name
	proxyInfo.ServerVersion = result.ServerInfo.Version

	// Store updated session
	session.SetProxyInfo(proxyInfo)

	return nil
}
