package proxy

import (
	"fmt"
	"net/http"

	"github.com/chatmcp/mcprouter/service/mcpserver"
	"github.com/chatmcp/mcprouter/service/proxy"
	"github.com/labstack/echo/v4"
)

// SSE is a handler for the sse endpoint
func SSE(c echo.Context) error {
	ctx, err := validateSSEContext(c)
	if err != nil {
		return err
	}

	// Parse and validate request
	key, serverConfig, err := parseSSERequest(c)
	if err != nil {
		return err
	}

	// Setup SSE connection and session
	writer, session, sessionID, err := setupSSEConnection(c, ctx, key, serverConfig)
	if err != nil {
		return err
	}
	defer ctx.DeleteSession(sessionID)

	// Handle SSE messages and lifecycle
	return handleSSEMessages(c, writer, session, sessionID)
}

// parseSSERequest validates request parameters and gets server config
func parseSSERequest(c echo.Context) (string, *mcpserver.ServerConfig, error) {
	// Use common validation function
	return validateKeyAndConfig(c)
}

// setupSSEConnection creates SSE writer, session and proxy info
func setupSSEConnection(c echo.Context, ctx *proxy.SSEContext, key string, serverConfig *mcpserver.ServerConfig) (*proxy.SSEWriter, *proxy.SSESession, string, error) {
	// Create SSE writer
	writer, err := proxy.NewSSEWriter(c)
	if err != nil {
		return nil, nil, "", c.String(http.StatusInternalServerError, err.Error())
	}

	// Create base proxy info using common function
	proxyInfo := createProxyInfo(key, serverConfig)

	// Generate session ID using common function and set it
	sessionID := proxyInfo.GetSessionID()
	proxyInfo.SessionID = sessionID

	// Create and store session
	session := proxy.NewSSESession(writer, serverConfig, proxyInfo)
	ctx.StoreSession(sessionID, session)

	return writer, session, sessionID, nil
}

// handleSSEMessages manages SSE message lifecycle and communication
func handleSSEMessages(c echo.Context, writer *proxy.SSEWriter, session *proxy.SSESession, sessionID string) error {
	req := c.Request()

	// Start session lifecycle goroutine
	go monitorSession(session, req)

	// Send endpoint information to client
	messagesUrl := fmt.Sprintf("/messages?sessionid=%s", sessionID)
	writer.SendEventData("endpoint", messagesUrl)

	// Main message handling loop
	for {
		select {
		case message := <-session.Messages():
			if err := writer.SendMessage(message); err != nil {
				fmt.Printf("SSE failed to send message to session %s: %v\n", sessionID, err)
				session.Close()
				return nil
			}
		case <-session.Done():
			fmt.Printf("Session %s closed\n", sessionID)
			return nil
		case <-req.Context().Done():
			fmt.Println("SSE request done")
			session.Close()
			return nil
		}
	}
}

// monitorSession monitors session and request lifecycle
func monitorSession(session *proxy.SSESession, req *http.Request) {
	for {
		select {
		case <-session.Done():
			return
		case <-req.Context().Done():
			return
		}
	}
}
