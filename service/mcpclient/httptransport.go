package mcpclient

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

type HttpTransport struct {
	c      *http.Client
	datach chan []byte
	url    string
}

func NewHttpTransport(url string) (*HttpTransport, error) {
	return &HttpTransport{
		c:      &http.Client{},
		datach: make(chan []byte, 10),
		url:    url,
	}, nil
}

func (t *HttpTransport) Write(p []byte) (n int, err error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, bytes.NewReader(p))
	if err != nil {
		return
	}
	request.Header.Set("content-type", "application/json")
	request.Header.Set("connection", "keep-alive")

	res, err := t.c.Do(request)
	if err != nil {
		return
	}

	go func() {
		defer res.Body.Close()

		reader := bufio.NewReader(res.Body)
		for {
			line, _, err := reader.ReadLine()
			if err != nil {
				break
			}
			t.datach <- append(line, []byte("\n")...)
		}
	}()

	return len(p), nil
}

func (t *HttpTransport) Close() error {
	return nil
}

func (t *HttpTransport) Read(p []byte) (n int, err error) {
	data := <-t.datach
	ln := copy(p, data)

	return ln, nil
}

func (t *HttpTransport) GetStdErrOutput() io.ReadCloser {
	return io.NopCloser(strings.NewReader(""))
}

func (t *HttpTransport) Start() {

}

func (t *HttpTransport) Stop() error {
	t.c.CloseIdleConnections()
	return nil
}
