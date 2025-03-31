package mcpclient

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/chatmcp/mcprouter/service/jsonrpc"
	"github.com/tidwall/gjson"
)

// StdioClient is a client that uses stdin and stdout to communicate with the backend mcp server.
type StdioClient struct {
	transport     Transport
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
func NewStdioClient(command string) (*StdioClient, error) {

	client, err := newStdioClient(command)
	if err != nil {
		return nil, err
	}
	err = client.transport.Start()
	if err != nil {
		return nil, err
	}

	stderr := client.transport.GetStdErrOutput()

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

	fmt.Printf("mcp server running with command: %s\n", command)

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

// SendMessage sends a JSON-RPC message to the MCP server and returns the response
func (c *StdioClient) SendMessage(message []byte) ([]byte, error) {
	// parsed message
	msg := gjson.ParseBytes(message)
	if msg.Get("jsonrpc").String() != jsonrpc.JSONRPC_VERSION {
		return nil, fmt.Errorf("invalid request message: %s", message)
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

	fmt.Printf("stdin write request message: %s\n", message)

	// wait for response
	for {
		select {
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

	return c.transport.Stop()
}
