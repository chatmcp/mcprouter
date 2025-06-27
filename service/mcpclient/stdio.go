package mcpclient

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/chatmcp/mcprouter/service/jsonrpc"
	"github.com/chatmcp/mcprouter/service/mcpserver"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// StdioClient is a client that uses stdin and stdout to communicate with the backend mcp server.
type StdioClient struct {
	serverConfig  *mcpserver.ServerConfig
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	stdout        *bufio.Reader
	stderr        *bufio.Reader
	done          chan struct{}         // client closed signal
	messages      map[int64]chan []byte // stdout messages channel
	mu            sync.RWMutex
	notifications []func(message []byte) // notification handlers
	nmu           sync.RWMutex
	err           chan error // stderr message
}

// NewStdioClient creates a new StdioClient.
func NewStdioClient(serverConfig *mcpserver.ServerConfig) (*StdioClient, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/C", serverConfig.Command)
	} else {
		cmd = exec.Command("sh", "-c", serverConfig.Command)
	}

	//cmd := exec.Command(
	//	"sh",
	//	"-c",
	//	serverConfig.Command,
	//)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	client := &StdioClient{
		serverConfig: serverConfig,
		cmd:          cmd,
		stdin:        stdin,
		stdout:       bufio.NewReader(stdout),
		stderr:       bufio.NewReader(stderr),
		done:         make(chan struct{}),
		messages:     make(map[int64]chan []byte),
		err:          make(chan error, 1),
	}

	// run command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// listen stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			errmsg := scanner.Text()
			fmt.Printf("stderr: %s\n", errmsg)
			// todo: detect error type, error or warning
			// only error message need to be sent
			// client.err <- fmt.Errorf("mcp server run failed: %s", errmsg)
			// client.Close()
			// return
		}

		if err := scanner.Err(); err != nil {
			client.err <- fmt.Errorf("stderr scanner error: %w", err)
			client.Close()
		}
	}()

	// listen stdout
	ready := make(chan struct{})
	go func() {
		close(ready)
		client.listen()
	}()
	<-ready

	fmt.Printf("mcp server running with command: %s\n", serverConfig.Command)

	return client, nil
}

// listen for messages from the backend mcp server.
func (c *StdioClient) listen() {
	for {
		select {
		case <-c.done:
			fmt.Println("client closed, cant read message")
			return

		default:
			message, err := c.stdout.ReadBytes('\n')
			// fmt.Printf("stdout read message: %s, %v\n", message, err)

			if err != nil {
				if err != io.EOF {
					fmt.Printf("failed to read message: %v\n", err)
				}
				c.Close()
				return
			}

			// parsed message
			msg := gjson.ParseBytes(message)
			if msg.Get("jsonrpc").String() != jsonrpc.JSONRPC_VERSION {
				fmt.Printf("invalid response message: %s\n", message)
				continue
			}

			// notification message
			if !msg.Get("id").Exists() {
				c.nmu.RLock()
				// send notification message to all handlers
				for _, handler := range c.notifications {
					handler(message)
				}
				c.nmu.RUnlock()
				continue
			}

			// not notification message
			id := msg.Get("id").Int()

			// result or error message
			c.mu.RLock()
			// get message channel
			msgch, ok := c.messages[id]
			c.mu.RUnlock()

			if !ok {
				// response message without corresponding request
				fmt.Printf("isolated response message: %s\n", message)
				continue
			}

			// send response message to channel
			msgch <- message
		}
	}
}

// Error returns the error message from stderr
func (c *StdioClient) Error() error {
	select {
	case err := <-c.err:
		return err
	default:
		return nil
	}
}

// Close client
func (c *StdioClient) Close() error {
	c.mu.Lock()
	select {
	case <-c.done:
		c.mu.Unlock()
		return nil
	default:
		close(c.done)
		c.mu.Unlock()
	}

	stdinClosed := make(chan error, 1)
	stdinWaiting := make(chan struct{})
	go func() {
		close(stdinWaiting)
		stdinClosed <- c.stdin.Close()
	}()
	<-stdinWaiting

	select {
	case err := <-stdinClosed:
		if err != nil {
			return fmt.Errorf("failed to close stdin: %w", err)
		}
	case <-time.After(3 * time.Second):
		return fmt.Errorf("timeout while closing stdin")
	}

	cmdClosed := make(chan error, 1)
	cmdWaiting := make(chan struct{})
	go func() {
		close(cmdWaiting)
		cmdClosed <- c.cmd.Wait()
	}()
	<-cmdWaiting

	select {
	case err := <-cmdClosed:
		return err
	case <-time.After(5 * time.Second):
		if err := c.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
		return fmt.Errorf("process killed after timeout")
	}
}

// OnNotification adds a notification handler
func (c *StdioClient) OnNotification(handler func(message []byte)) {
	c.nmu.Lock()
	c.notifications = append(c.notifications, handler)
	c.nmu.Unlock()
}

// SendMessage sends a JSON-RPC message to the MCP server and returns the response
func (c *StdioClient) SendMessage(message []byte) ([]byte, error) {
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

	message = append(message, '\n')

	if !msg.Get("id").Exists() {
		// notification message
		if _, err := c.stdin.Write(message); err != nil {
			return nil, fmt.Errorf("failed to write notification message: %w", err)
		}

		fmt.Printf("stdin write notification message: %s\n", message)

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

	if _, err := c.stdin.Write(message); err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to write request message: %w", err)
	}

	// fmt.Printf("stdin write request message: %s\n", message)

	timeout := time.After(30 * time.Second)

	// wait for response
	for {
		select {
		case <-timeout:
			fmt.Println("timeout waiting for response after 30 seconds")
			c.Close()
			return nil, fmt.Errorf("timeout waiting for response after 30 seconds")
		case <-c.done:
			fmt.Println("client closed with no response")
			return nil, fmt.Errorf("client closed with no response")
		case err := <-c.err:
			fmt.Printf("stderr with no response: %s\n", err)
			return nil, err
		case response := <-msgch:
			return response, nil
		}
	}
}

// ForwardMessage forwards a JSON-RPC message to the MCP server and returns the response
func (c *StdioClient) ForwardMessage(request *jsonrpc.Request) (*jsonrpc.Response, error) {
	// fmt.Printf("forward request: %+v\n", request)

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

	response := &jsonrpc.Response{}
	if err := json.Unmarshal(res, response); err != nil {
		return nil, err
	}

	// fmt.Printf("forward response: %+v\n", response)

	return response, nil
}

// Initialize initializes the client.
func (c *StdioClient) Initialize(params *jsonrpc.InitializeParams) (*jsonrpc.InitializeResult, error) {
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
func (c *StdioClient) NotificationsInitialized() error {
	request := jsonrpc.NewRequest(jsonrpc.MethodInitializedNotification, nil, nil)

	_, err := c.ForwardMessage(request)
	if err != nil {
		return err
	}

	return nil
}

// ListTools lists the tools available in the MCP server.
func (c *StdioClient) ListTools() (*jsonrpc.ListToolsResult, error) {
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
func (c *StdioClient) CallTool(params *jsonrpc.CallToolParams) (*jsonrpc.CallToolResult, error) {
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
