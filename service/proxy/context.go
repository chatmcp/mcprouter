package proxy

import (
	"io"
	"net/http"
	"sync"

	"github.com/chatmcp/mcprouter/service/jsonrpc"
	"github.com/chatmcp/mcprouter/service/mcpclient"
	"github.com/labstack/echo/v4"
)

// SSEContext is the context for SSE request
type SSEContext struct {
	echo.Context
	sessions *sync.Map // sessions store
	clients  *sync.Map // clients store
}

// createSSEMiddleware will create a middleware for http request
func createSSEMiddleware(sessions *sync.Map, clients *sync.Map) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := &SSEContext{
				Context:  c,
				sessions: sessions,
				clients:  clients,
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

// StoreClient stores the client in the clients store
func (c *SSEContext) StoreClient(key string, client mcpclient.Client) {
	c.clients.Store(key, client)
}

// GetClient returns the client from the clients store
func (c *SSEContext) GetClient(key string) mcpclient.Client {
	if client, ok := c.clients.Load(key); ok {
		return client.(mcpclient.Client)
	}

	return nil
}

// DeleteClient deletes the client from the clients store
func (c *SSEContext) DeleteClient(key string) {
	if client, ok := c.clients.Load(key); ok {
		client.(mcpclient.Client).Close()
	}

	c.clients.Delete(key)
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
	return c.JSON(http.StatusOK, response)
}

// JSONRPCAcceptResponse returns a JSON-RPC accept response
func (c *SSEContext) JSONRPCAcceptResponse(response *jsonrpc.Response) error {
	return c.JSON(http.StatusAccepted, response)
}

// JSONRPCStreamResponse returns a JSON-RPC response in SSE format for streaming
func (c *SSEContext) JSONRPCStreamResponse(response *jsonrpc.Response) error {
	writer, err := NewSSEWriter(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "SSE not supported"})
	}

	// Send the JSON-RPC response as SSE data
	responseJSON := response.String()
	if err := writer.SendEventData("jsonrpc", responseJSON); err != nil {
		return err
	}

	// Send completion event to indicate end of response
	if err := writer.SendEventData("stream", "completed"); err != nil {
		return err
	}

	// Return nil to indicate successful handling
	return nil
}

// JSONRPCStreamStart initiates a streaming response and returns the writer for continued streaming
func (c *SSEContext) JSONRPCStreamStart() (*SSEWriter, error) {
	writer, err := NewSSEWriter(c)
	if err != nil {
		return nil, err
	}

	// Send initial event to indicate streaming has started
	err = writer.SendEventData("stream", "started")
	if err != nil {
		return nil, err
	}

	return writer, nil
}

// JSONRPCStreamComplete sends a completion event and closes the stream
func (c *SSEContext) JSONRPCStreamComplete(writer *SSEWriter, response *jsonrpc.Response) error {
	if response != nil {
		responseJSON := response.String()
		if err := writer.SendEventData("jsonrpc", responseJSON); err != nil {
			return err
		}
	}

	// Send completion event
	return writer.SendEventData("stream", "completed")
}
