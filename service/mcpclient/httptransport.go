package mcpclient

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HttpTransport struct {
	baseURL      *url.URL
	endpoint     *url.URL
	httpClient   *http.Client
	messageCh    chan []byte
	done         chan struct{}
	endpointChan chan struct{}
}

func NewHttpTransport(baseURL string) (*HttpTransport, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	ht := &HttpTransport{
		baseURL:      parsedURL,
		httpClient:   &http.Client{},
		messageCh:    make(chan []byte, 10),
		done:         make(chan struct{}),
		endpointChan: make(chan struct{}),
	}
	return ht, nil
}

func (t *HttpTransport) Write(p []byte) (n int, err error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, t.endpoint.String(), bytes.NewReader(p))
	if err != nil {
		return 0, err
	}
	request.Header.Set("content-type", "application/json")

	resp, err := t.httpClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf(
			"request failed with status %d: %s",
			resp.StatusCode,
			body,
		)
	}

	return len(p), nil
}

func (t *HttpTransport) Close() error {
	return nil
}

func (t *HttpTransport) Read(p []byte) (n int, err error) {
	data := <-t.messageCh
	ln := copy(p, data)

	return ln, nil
}

func (t *HttpTransport) GetStdErrOutput() io.ReadCloser {
	return io.NopCloser(strings.NewReader(""))
}

func (t *HttpTransport) Start() error {

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, t.baseURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to SSE stream: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	go t.readSSE(resp.Body)

	// Wait for the endpoint to be received
	select {
	case <-t.endpointChan:
		// Endpoint received, proceed
	case <-time.After(30 * time.Second): // Add a timeout
		return fmt.Errorf("timeout waiting for endpoint")
	}

	return nil
}

func (t *HttpTransport) Stop() error {
	close(t.done)
	return nil
}

func (t *HttpTransport) readSSE(reader io.ReadCloser) {
	defer reader.Close()

	br := bufio.NewReader(reader)
	var event, data string

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Process any pending event before exit
				if event != "" && data != "" {
					t.handleSSEEvent(event, data)
				}
				break
			}
			select {
			case <-t.done:
				return
			default:
				fmt.Printf("SSE stream error: %v\n", err)
				return
			}
		}

		// Remove only newline markers
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			// Empty line means end of event
			if event != "" && data != "" {
				t.handleSSEEvent(event, data)
				event = ""
				data = ""
			}
			continue
		}

		if strings.HasPrefix(line, "event:") {
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}
}

func (t *HttpTransport) handleSSEEvent(event, data string) {
	switch event {
	case "endpoint":
		endpoint, err := t.baseURL.Parse(data)
		if err != nil {
			fmt.Printf("Error parsing endpoint URL: %v\n", err)
			return
		}
		if endpoint.Host != t.baseURL.Host {
			fmt.Printf("Endpoint origin does not match connection origin\n")
			return
		}
		t.endpoint = endpoint
		close(t.endpointChan)

	case "message":
		select {
		case t.messageCh <- []byte(data + "\n"):
		default:
			fmt.Println("message cache overflow", string(data))
		}
	}
}
