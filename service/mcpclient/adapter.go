package mcpclient

import (
	"bufio"
	"io"
	"strings"
)

type Transport interface {
	io.ReadWriteCloser
	GetStdErrOutput() io.ReadCloser
	Start()
	Stop() error
}

func newStdioClient(command string) (*StdioClient, error) {
	var transport Transport
	var err error
	if strings.HasPrefix(command, "http://") {
		transport, err = NewHttpTransport(command)
	} else {
		transport, err = NewStdioTransport(command)
	}
	if err != nil {
		return nil, err
	}

	client := &StdioClient{
		transport: transport,
		stdin:     transport,
		stdout:    bufio.NewReader(transport),
		stderr:    bufio.NewReader(transport),
		done:      make(chan struct{}),
		messages:  make(map[int64]chan []byte),
		err:       make(chan error, 1),
	}

	return client, nil
}
