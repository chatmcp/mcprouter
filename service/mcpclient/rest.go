package mcpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chatmcp/mcprouter/service/jsonrpc"
	"github.com/chatmcp/mcprouter/service/mcpserver"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// RestClient is a client that uses HTTP to communicate with the backend mcp server.
type RestClient struct {
	serverConfig  *mcpserver.ServerConfig
	httpClient    *http.Client
	done          chan struct{}         // client closed signal
	messages      map[int64]chan []byte // response messages channel
	mu            sync.RWMutex
	notifications []func(message []byte) // notification handlers
	nmu           sync.RWMutex
	err           chan error // error channel
}

// NewRestClient creates a new RestClient.
func NewRestClient(serverConfig *mcpserver.ServerConfig) (*RestClient, error) {
	client := &RestClient{
		serverConfig: serverConfig,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		done:         make(chan struct{}),
		messages:     make(map[int64]chan []byte),
		err:          make(chan error, 1),
	}

	fmt.Printf("mcp server connecting to: %s\n", serverConfig.ServerURL)

	return client, nil
}

// Error returns the error message
func (c *RestClient) Error() error {
	select {
	case err := <-c.err:
		return err
	default:
		return nil
	}
}

// Close client
func (c *RestClient) Close() error {
	c.mu.Lock()
	select {
	case <-c.done:
		c.mu.Unlock()
		return nil
	default:
		close(c.done)
		c.mu.Unlock()
	}

	// cancel any pending requests by setting a short timeout
	c.httpClient.Timeout = 100 * time.Millisecond

	return nil
}

// OnNotification adds a notification handler
func (c *RestClient) OnNotification(handler func(message []byte)) {
	c.nmu.Lock()
	c.notifications = append(c.notifications, handler)
	c.nmu.Unlock()
}

// SendMessage sends a JSON-RPC message to the MCP server and returns the response
func (c *RestClient) SendMessage(message []byte) ([]byte, error) {
	fmt.Printf("sending message: %s\n", message)

	// parsed message
	msg := gjson.ParseBytes(message)
	if msg.Get("jsonrpc").String() != jsonrpc.JSONRPC_VERSION {
		return nil, fmt.Errorf("invalid request message: %s", message)
	}

	serverParams := map[string]interface{}{}
	if c.serverConfig.ServerParams != "" {
		if err := json.Unmarshal([]byte(c.serverConfig.ServerParams), &serverParams); err != nil {
			fmt.Printf("failed to unmarshal server params: %v\n", err)
		}
	}

	metadata := map[string]interface{}{
		"auth": serverParams,
	}

	var err error

	message, err = sjson.SetBytes(message, "params._meta", metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to modify message: %w", err)
	}

	if msg.Get("method").String() == "initialize" {
		message, err = sjson.SetBytes(message, "params.capabilities.experimental", metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to modify initialize message: %w", err)
		}
	}

	if !msg.Get("id").Exists() {
		// notification message
		req, err := http.NewRequest("POST", c.serverConfig.ServerURL, bytes.NewReader(message))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send notification: %w", err)
		}
		defer resp.Body.Close()

		fmt.Printf("sent notification message: %s\n", message)

		return nil, nil
	}

	// not notification message
	id := msg.Get("id").Int()

	// message channel
	msgch := make(chan []byte, 1)

	c.mu.Lock()
	c.messages[id] = msgch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.messages, id)
		c.mu.Unlock()
	}()

	req, err := http.NewRequest("POST", c.serverConfig.ServerURL, bytes.NewReader(message))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Add debug information
	fmt.Printf("HTTP Status: %d\n", resp.StatusCode)
	fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("Response Body: %s\n", string(responseBody))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status code %d: %s", resp.StatusCode, responseBody)
	}

	// Check response type by Content-Type header
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		// Parse SSE format and extract data field
		responseStr := string(responseBody)
		lines := strings.Split(responseStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data: ") {
				// Extract content from data field
				dataContent := strings.TrimPrefix(line, "data: ")
				return []byte(dataContent), nil
			}
		}
		// If no data field found, return original response
		return responseBody, nil
	}

	return responseBody, nil
}

// ForwardMessage forwards a JSON-RPC message to the MCP server and returns the response
func (c *RestClient) ForwardMessage(request *jsonrpc.Request) (*jsonrpc.Response, error) {
	req, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	res, err := c.SendMessage(req)
	if err != nil {
		fmt.Printf("failed to forward message: %v\n", err)
		return nil, err
	}

	// notification message with no response
	if res == nil {
		return nil, nil
	}

	// Add debug information
	fmt.Printf("Attempting to parse response: %s\n", string(res))

	response := &jsonrpc.Response{}
	if err := json.Unmarshal(res, response); err != nil {
		return nil, err
	}

	return response, nil
}

// Initialize initializes the client.
func (c *RestClient) Initialize(params *jsonrpc.InitializeParams) (*jsonrpc.InitializeResult, error) {
	request := jsonrpc.NewRequest(jsonrpc.MethodInitialize, params, 0)

	response, err := c.ForwardMessage(request)
	if err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, response.Error
	}

	result := &jsonrpc.InitializeResult{}
	if err := response.UnmarshalResult(result); err != nil {
		return nil, err
	}

	return result, nil
}

// NotificationsInitialized sends the initialized notification to the server.
func (c *RestClient) NotificationsInitialized() error {
	request := jsonrpc.NewRequest(jsonrpc.MethodInitializedNotification, nil, nil)

	_, err := c.ForwardMessage(request)
	if err != nil {
		return err
	}

	return nil
}

// ListTools lists the tools available in the MCP server.
func (c *RestClient) ListTools() (*jsonrpc.ListToolsResult, error) {
	request := jsonrpc.NewRequest(jsonrpc.MethodListTools, nil, 1)

	response, err := c.ForwardMessage(request)
	if err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, response.Error
	}

	result := &jsonrpc.ListToolsResult{}
	if err := response.UnmarshalResult(result); err != nil {
		return nil, err
	}

	return result, nil
}

// CallTool calls a tool with the given name and arguments.
func (c *RestClient) CallTool(params *jsonrpc.CallToolParams) (*jsonrpc.CallToolResult, error) {
	request := jsonrpc.NewRequest(jsonrpc.MethodCallTool, params, 1)

	response, err := c.ForwardMessage(request)
	if err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, response.Error
	}

	result := &jsonrpc.CallToolResult{}
	if err := response.UnmarshalResult(result); err != nil {
		return nil, err
	}

	return result, nil
}
