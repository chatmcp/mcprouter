package proxy

import (
	"fmt"
	"net/http"
	"time"

	"github.com/chatmcp/mcprouter/service/mcpserver"
	"github.com/chatmcp/mcprouter/service/proxy"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// SSE is a handler for the sse endpoint
func SSE(c echo.Context) error {
	ctx := proxy.GetSSEContext(c)
	if ctx == nil {
		return c.String(http.StatusInternalServerError, "Failed to get SSE context")
	}

	req := c.Request()

	key := c.Param("key")
	if key == "" {
		return c.String(http.StatusBadRequest, "Key is required")
	}

	config := mcpserver.GetConfig(key)
	if config == nil {
		return c.String(http.StatusBadRequest, "Server command not found")
	}

	writer, err := proxy.NewSSEWriter(c)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	// store session
	sessionID := uuid.New().String()
	session := proxy.NewSSESession(writer, key, config)
	ctx.StoreSession(sessionID, session)
	defer ctx.DeleteSession(sessionID)

	// Setup heartbeat ticker
	heartbeatInterval := 30 * time.Second // adjust interval as needed
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	// Setup idle timeout
	idleTimeout := 5 * time.Minute // adjust timeout as needed
	idleTimer := time.NewTimer(idleTimeout)
	defer idleTimer.Stop()

	// Reset idle timer when activity occurs
	resetIdleTimer := func() {
		if !idleTimer.Stop() {
			<-idleTimer.C
		}
		idleTimer.Reset(idleTimeout)
	}

	go func() {
		for {
			select {
			case <-session.Done():
				return
			case <-req.Context().Done():
				return
			case <-heartbeatTicker.C:
				// Send heartbeat comment
				if err := writer.SendHeartbeat(); err != nil {
					session.Close()
					return
				}
			case <-idleTimer.C:
				// Close connection due to inactivity
				session.Close()
				return
			}
		}
	}()

	// response to client with endpoint url
	messagesUrl := fmt.Sprintf("/messages?sessionid=%s", sessionID)
	writer.SendEventData("endpoint", messagesUrl)

	// listen to messages
	for {
		select {
		case message := <-session.Messages():
			// Reset idle timer on message activity
			resetIdleTimer()

			if err := writer.SendMessage(message); err != nil {
				fmt.Printf("sse failed to send message to session %s: %v\n", sessionID, err)
				session.Close() // Close session on send error
				return nil      // Exit the handler
			}
		case <-session.Done():
			fmt.Printf("session %s closed \n", sessionID)
			return nil
		case <-req.Context().Done():
			fmt.Println("sse request done")
			session.Close()
			return nil
		}
	}
}
