package mcpclient

import (
	"crypto/md5"
	"fmt"
	"github.com/chatmcp/mcprouter/service/mcpserver"
	"sync"
)

var (
	clientPool = sync.Map{}
	clients    = sync.Map{}
)

func GetStdioClient(config *mcpserver.Config) (*StdioClient, error) {
	if config.ShareProcess {
		//md5 config.CMD
		hash := md5.Sum([]byte(config.CMD))
		if v, ok := clientPool.Load(hash); ok {
			return v.(*StdioClient), nil
		}
		if c, err := newStdioClient(config, func(c *StdioClient) {
			clientPool.Delete(hash)
		}); nil == err {
			clientPool.Store(hash, c)
			return c, nil
		} else {
			fmt.Errorf("failed to create stdio client: %w", err)
			return nil, err
		}
	}

	if c, ok := clients.Load(config.Key); ok {
		return c.(*StdioClient), nil
	}

	if c, err := newStdioClient(config, func(c *StdioClient) {
		clients.Delete(config.Key)
	}); nil == err {
		clients.Store(config.Key, c)
		return c, nil
	} else {
		fmt.Errorf("failed to create stdio client: %w", err)
		return nil, err
	}
}
