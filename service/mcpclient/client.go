package mcpclient

import (
	"fmt"
	"log"
	"strings"

	"github.com/chatmcp/mcprouter/service/jsonrpc"
	"github.com/chatmcp/mcprouter/service/mcpserver"
)

// Client is a client that can send and receive messages to and from the server.
type Client interface {
	Error() error
	Close() error
	OnNotification(handler func(message []byte))
	SendMessage(message []byte) ([]byte, error)
	ForwardMessage(request *jsonrpc.Request) (*jsonrpc.Response, error)
	Initialize(params *jsonrpc.InitializeParams) (*jsonrpc.InitializeResult, error)
	NotificationsInitialized() error
	ListTools() (*jsonrpc.ListToolsResult, error)
	CallTool(params *jsonrpc.CallToolParams) (*jsonrpc.CallToolResult, error)
}

// NewClient creates a new client
func NewClient(serverConfig *mcpserver.ServerConfig) (Client, error) {
	log.Printf("new client with server config: %+v\n", serverConfig)

	if serverConfig.ServerURL != "" {
		if !strings.HasPrefix(serverConfig.ServerURL, "http") {
			return nil, fmt.Errorf("invalid server url")
		}

		return NewRestClient(serverConfig)
	}

	if serverConfig.Command == "" {
		return nil, fmt.Errorf("invalid command")
	}

	return NewStdioClient(serverConfig)
}
