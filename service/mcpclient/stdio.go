package mcpclient

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/chatmcp/mcprouter/service/jsonrpc"
	"github.com/chatmcp/mcprouter/service/mcpserver"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var (
	gloableId atomic.Int64
	idMapping = sync.Map{} // ID map store the mapping between global ID and original ID
)

func getGlobalStringId() string {
	return uuid.New().String()
	//gloableId.Add(1)
	//return gloableId.Load()
}

type OnClientClosed func(client *StdioClient)

// StdioClient is a client that uses stdin and stdout to communicate with the backend mcp server.
type StdioClient struct {
	config        *mcpserver.Config
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	stdout        *bufio.Reader
	stderr        *bufio.Reader
	done          chan struct{} // client closed signal
	messages      sync.Map      // stdout messages channel
	mu            sync.RWMutex
	notifications []func(message []byte) // notification handlers
	nmu           sync.RWMutex
	err           chan error // stderr message
	closedFunc    OnClientClosed
}

// NewStdioClient creates a new StdioClient.
func newStdioClient(config *mcpserver.Config, onClientClosed OnClientClosed) (*StdioClient, error) {
	// check if  the server can  share process

	cmd := exec.Command(
		"sh",
		"-c",
		config.CMD,
	)

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
		config:     config,
		cmd:        cmd,
		stdin:      stdin,
		stdout:     bufio.NewReader(stdout),
		stderr:     bufio.NewReader(stderr),
		done:       make(chan struct{}),
		messages:   sync.Map{},
		err:        make(chan error, 1),
		closedFunc: onClientClosed,
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

	fmt.Printf("mcp server running with command: %s\n", config.CMD)
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
			fmt.Printf("stdout read message: %s, %v\n", message, err)

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
			messageId := msg.Get("id").Value()
			var newMessage []byte
			//replace id to original id
			if id, ok := idMapping.Load(messageId); ok {
				if s, err := sjson.Set(string(message), "id", id); nil == err {
					newMessage = []byte(s)
				} else {
					newMessage = message
				}
				idMapping.Delete(messageId)
			} else {
				newMessage = message
			}
			if msgch, ok := c.messages.Load(messageId); ok {
				msgch.(chan []byte) <- newMessage
			} else {
				// response message without corresponding request
				fmt.Printf("isolated response message: %s\n", message)
				continue
			}
		}
	}
}

// SendMessage sends a JSON-RPC message to the MCP server and returns the response
func (c *StdioClient) SendMessage(message []byte) ([]byte, error) {
	fmt.Printf("stdin write request Message: %s\n", message)
	// parsed message
	msg := gjson.ParseBytes(message)
	if msg.Get("jsonrpc").String() != jsonrpc.JSONRPC_VERSION {
		return nil, fmt.Errorf("invalid request message: %s", message)
	}

	if !msg.Get("id").Exists() {
		// notification message
		if _, err := c.stdin.Write(message); err != nil {
			return nil, fmt.Errorf("failed to write notification message: %w", err)
		}

		fmt.Printf("stdin write notification message: %s\n", message)
		return nil, nil
	}
	// message channel
	messageChannel := make(chan []byte, 1)
	id := msg.Get("id").Value()
	var newMessage []byte

	var newId = getGlobalStringId()
	c.messages.Store(newId, messageChannel)
	idMapping.Store(newId, id)
	defer func() {
		if _, ok := c.messages.Load(newId); ok {
			c.messages.Delete(newId)
		}
	}()
	if s, err := sjson.Set(msg.String(), "id", newId); nil == err {
		newMessage = []byte(s)
	} else {
		return nil, fmt.Errorf("set id error")
	}

	newMessage = append(newMessage, '\n')
	if _, err := c.stdin.Write(newMessage); err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to write request newMessage: %w", err)
	}

	fmt.Printf("stdin write request newMessage: %s\n", newMessage)

	// wait for response
	for {
		select {
		case <-c.done:
			fmt.Println("client closed with no response")
			//delete from pool,if the client was closed
			if c.closedFunc != nil {
				c.closedFunc(c)
			}
			return nil, fmt.Errorf("client closed with no response")
		case err := <-c.err:
			fmt.Printf("stderr with no response: %s\n", err)
			return nil, err
		case response := <-messageChannel:
			return response, nil
		}
	}
}

// OnNotification adds a notification handler
func (c *StdioClient) OnNotification(handler func(message []byte)) {
	c.nmu.Lock()
	c.notifications = append(c.notifications, handler)
	c.nmu.Unlock()
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
		// Channel is already closed
		c.mu.Unlock()
		return nil
	default:
		close(c.done)
		c.mu.Unlock()
	}

	if err := c.stdin.Close(); err != nil {
		return fmt.Errorf("failed to close stdin: %w", err)
	}

	if nil != c.closedFunc {
		c.closedFunc(c)
	}
	return c.cmd.Wait()
}
