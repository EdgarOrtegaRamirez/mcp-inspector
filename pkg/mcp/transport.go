package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// Transport is the interface for communicating with MCP servers
type Transport interface {
	SendRequest(ctx context.Context, req Request) (*Response, error)
	SendNotification(ctx context.Context, notif Notification) error
	ReceiveMessage(ctx context.Context) (interface{}, error)
	Close() error
}

// StdioTransport communicates with an MCP server over stdin/stdout
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	mu      sync.Mutex
	id      int
	pending map[interface{}]chan *Response
}

// NewStdioTransport creates a new stdio transport for an MCP server
func NewStdioTransport(command string, args ...string) (*StdioTransport, error) {
	cmd := exec.Command(command, args...)
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

	return &StdioTransport{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		pending: make(map[interface{}]chan *Response),
	}, nil
}

// Start starts the MCP server process
func (t *StdioTransport) Start() error {
	return t.cmd.Start()
}

// SendRequest sends a request and waits for a response
func (t *StdioTransport) SendRequest(ctx context.Context, req Request) (*Response, error) {
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

	// MCP uses newline-delimited JSON
	data = append(data, '\n')

	t.mu.Lock()
	_, err = t.stdin.Write(data)
	t.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// SendNotification sends a notification (no response expected)
func (t *StdioTransport) SendNotification(ctx context.Context, notif Notification) error {
	data, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	data = append(data, '\n')

	t.mu.Lock()
	_, err = t.stdin.Write(data)
	t.mu.Unlock()
	return err
}

// ReceiveMessage reads the next message from the server
func (t *StdioTransport) ReceiveMessage(ctx context.Context) (interface{}, error) {
	reader := bufio.NewReader(t.stdout)

	line, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	// Try parsing as response first
	var resp Response
	if err := json.Unmarshal(line, &resp); err == nil && resp.ID != nil {
		t.mu.Lock()
		if ch, ok := t.pending[resp.ID]; ok {
			ch <- &resp
			t.mu.Unlock()
			return &resp, nil
		}
		t.mu.Unlock()
		return &resp, nil
	}

	// Try parsing as notification
	var notif Notification
	if err := json.Unmarshal(line, &notif); err == nil && notif.Method != "" {
		return &notif, nil
	}

	return nil, fmt.Errorf("unknown message format: %s", string(line))
}

// StartReading starts reading messages in the background
func (t *StdioTransport) StartReading(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, err := t.ReceiveMessage(ctx)
				if err != nil {
					return
				}
			}
		}
	}()
}

// Close closes the transport
func (t *StdioTransport) Close() error {
	if t.stdin != nil {
		t.stdin.Close()
	}
	if t.cmd != nil && t.cmd.Process != nil {
		return t.cmd.Process.Kill()
	}
	return nil
}
