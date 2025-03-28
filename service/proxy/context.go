package proxy

import (
	"io"
	"net/http"
	"sync"

	"github.com/chatmcp/mcprouter/service/jsonrpc"
	"github.com/labstack/echo/v4"
)

// SSEContext is the context for SSE request
type SSEContext struct {
	echo.Context
	sessions *sync.Map // sessions store
	clients  *sync.Map
}

// createSSEMiddleware will create a middleware for http request
func createSSEMiddleware(sessions *sync.Map) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := &SSEContext{
				Context:  c,
				sessions: sessions,
			}

			return next(ctx)
		}
	}
}

// GetSSEContext returns the SSEContext from the echo.Context
func GetSSEContext(c echo.Context) *SSEContext {
	if ctx, ok := c.(*SSEContext); ok {
		return ctx
	}

	return nil
}

// GetSession returns the session from the sessions store
func (c *SSEContext) GetSession(key string) *SSESession {
	if session, ok := c.sessions.Load(key); ok {
		return session.(*SSESession)
	}

	return nil
}

// StoreSession stores the session in the sessions store
func (c *SSEContext) StoreSession(key string, session *SSESession) {
	c.sessions.Store(key, session)
}

// DeleteSession deletes the session from the sessions store
func (c *SSEContext) DeleteSession(key string) {
	c.sessions.Delete(key)
}

// GetJSONRPCRequest returns the JSON-RPC request from the request body
func (c *SSEContext) GetJSONRPCRequest() (*jsonrpc.Request, error) {
	req := c.Request()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	request, err := jsonrpc.UnmarshalRequest(body)
	if err != nil {
		return nil, err
	}

	return request, nil
}

// JSONRPCError returns a JSON-RPC error response
func (c *SSEContext) JSONRPCError(err *jsonrpc.Error, id interface{}) error {
	response := jsonrpc.NewErrorResponse(err, id)

	return c.JSON(http.StatusBadRequest, response)
}

// JSONRPCResponse returns a JSON-RPC response
func (c *SSEContext) JSONRPCResponse(response *jsonrpc.Response) error {
	return c.JSON(http.StatusAccepted, response)
}
