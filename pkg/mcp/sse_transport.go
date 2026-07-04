package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// SSETransport communicates with an MCP server over HTTP Server-Sent Events
type SSETransport struct {
	endpoint string
	client   *http.Client

	mu      sync.Mutex
	id      int
	pending map[interface{}]chan *Response
	eventCh chan SSEEvent
	done    chan struct{}
}

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Event string
	Data  string
	ID    string
}

// NewSSETransport creates a new SSE transport
func NewSSETransport(endpoint string) *SSETransport {
	return &SSETransport{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		pending: make(map[interface{}]chan *Response),
		eventCh: make(chan SSEEvent, 100),
		done:    make(chan struct{}),
	}
}

// Connect establishes the SSE connection
func (t *SSETransport) Connect(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", t.endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	go t.readSSE(resp.Body)
	return nil
}

func (t *SSETransport) readSSE(body io.ReadCloser) {
	defer close(t.done)
	defer body.Close()

	reader := newSSEReader(body)
	for {
		event, err := reader.ReadEvent()
		if err != nil {
			return
		}
		t.eventCh <- event
	}
}

// SendRequest sends a request and waits for a response via SSE
func (t *SSETransport) SendRequest(ctx context.Context, req Request) (*Response, error) {
	t.mu.Lock()
	reqID := req.ID
	ch := make(chan *Response, 1)
	t.pending[reqID] = ch
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		delete(t.pending, reqID)
		t.mu.Unlock()
	}()

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send as POST to the message endpoint
	msgURL := t.endpoint + "/message"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", msgURL, bytesReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("unexpected status: %d", httpResp.StatusCode)
	}

	// Wait for response via SSE
	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// SendNotification sends a notification (no response expected)
func (t *SSETransport) SendNotification(ctx context.Context, notif Notification) error {
	data, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	msgURL := t.endpoint + "/message"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", msgURL, bytesReader(data))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// ReceiveMessage reads the next message
func (t *SSETransport) ReceiveMessage(ctx context.Context) (interface{}, error) {
	select {
	case event := <-t.eventCh:
		return t.processSSEEvent(event)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (t *SSETransport) processSSEEvent(event SSEEvent) (interface{}, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(event.Data), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse SSE data: %w", err)
	}

	// Check if it has an "id" field (response)
	if _, ok := raw["id"]; ok {
		var resp Response
		if err := json.Unmarshal([]byte(event.Data), &resp); err == nil {
			t.mu.Lock()
			if ch, ok := t.pending[resp.ID]; ok {
				ch <- &resp
			}
			t.mu.Unlock()
			return &resp, nil
		}
	}

	// Check if it has a "method" field (notification)
	if _, ok := raw["method"]; ok {
		var notif Notification
		if err := json.Unmarshal([]byte(event.Data), &notif); err == nil {
			return &notif, nil
		}
	}

	return raw, nil
}

// Close closes the transport
func (t *SSETransport) Close() error {
	close(t.done)
	return nil
}

// bytesReader creates a reader from a byte slice
func bytesReader(data []byte) io.Reader {
	return &bytesReaderImpl{data: data, pos: 0}
}

type bytesReaderImpl struct {
	data []byte
	pos  int
}

func (r *bytesReaderImpl) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// SSE reader
type sseReader struct {
	reader io.Reader
}

func newSSEReader(r io.Reader) *sseReader {
	return &sseReader{reader: r}
}

func (r *sseReader) ReadEvent() (SSEEvent, error) {
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 1)
	event := SSEEvent{}

	for {
		n, err := r.reader.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[0])
			if tmp[0] == '\n' {
				line := string(buf[:len(buf)-1])
				buf = buf[:0]

				if line == "" {
					// Empty line = end of event
					if event.Data != "" {
						return event, nil
					}
					continue
				}

				if len(line) > 6 && line[:6] == "event:" {
					event.Event = line[6:]
				} else if len(line) > 5 && line[:5] == "data:" {
					event.Data = line[5:]
				} else if len(line) > 3 && line[:3] == "id:" {
					event.ID = line[3:]
				}
			}
		}
		if err != nil {
			return event, err
		}
	}
}
