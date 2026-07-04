package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestNewRequest(t *testing.T) {
	req := NewRequest(1, "initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
	})

	if req.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %s", req.JSONRPC)
	}
	if req.ID != 1 {
		t.Errorf("expected id 1, got %v", req.ID)
	}
	if req.Method != "initialize" {
		t.Errorf("expected method initialize, got %s", req.Method)
	}
}

func TestRequestMarshalJSON(t *testing.T) {
	req := NewRequest(1, "test", nil)
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed["jsonrpc"] != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %v", parsed["jsonrpc"])
	}
	if parsed["method"] != "test" {
		t.Errorf("expected method test, got %v", parsed["method"])
	}
}

func TestParseResponse(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1,"result":{"name":"test"}}`)
	resp, err := ParseResponse(data)
	if err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %s", resp.JSONRPC)
	}
	if resp.Error != nil {
		t.Errorf("expected no error, got %v", resp.Error)
	}
}

func TestParseResponseWithError(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`)
	resp, err := ParseResponse(data)
	if err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("expected error code -32601, got %d", resp.Error.Code)
	}
}

func TestParseNotification(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	notif, err := ParseNotification(data)
	if err != nil {
		t.Fatalf("failed to parse notification: %v", err)
	}

	if notif.Method != "notifications/initialized" {
		t.Errorf("expected method notifications/initialized, got %s", notif.Method)
	}
}

func TestClientInitialize(t *testing.T) {
	// This is a basic test - in production, you'd mock the transport
	transport := &mockTransport{
		responses: map[string]*Response{
			"1": {
				JSONRPC: "2.0",
				ID:      1,
				Result: map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities": map[string]interface{}{
						"tools": map[string]interface{}{},
					},
					"serverInfo": map[string]interface{}{
						"name":    "test-server",
						"version": "1.0.0",
					},
				},
			},
		},
	}

	client := NewClient(transport)
	ctx := context.Background()

	caps, err := client.Initialize(ctx)
	if err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	if caps.Info.Name != "test-server" {
		t.Errorf("expected server name test-server, got %s", caps.Info.Name)
	}
	if caps.Info.Version != "1.0.0" {
		t.Errorf("expected server version 1.0.0, got %s", caps.Info.Version)
	}
	if caps.Capabilities.Tools == nil {
		t.Error("expected tools capability")
	}
}

func TestToolTypes(t *testing.T) {
	tool := Tool{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	if tool.Name != "test-tool" {
		t.Errorf("expected name test-tool, got %s", tool.Name)
	}
	if tool.Description != "A test tool" {
		t.Errorf("expected description A test tool, got %s", tool.Description)
	}
	if tool.InputSchema == nil {
		t.Error("expected input schema")
	}
}

func TestResourceTypes(t *testing.T) {
	resource := Resource{
		URI:         "file:///test.txt",
		Name:        "test.txt",
		Description: "A test file",
		MimeType:    "text/plain",
	}

	if resource.URI != "file:///test.txt" {
		t.Errorf("expected URI file:///test.txt, got %s", resource.URI)
	}
}

// mockTransport is a mock implementation of Transport for testing
type mockTransport struct {
	responses map[string]*Response
	closed    bool
}

func (m *mockTransport) SendRequest(ctx context.Context, req Request) (*Response, error) {
	idStr := ""
	if id, ok := req.ID.(int); ok {
		idStr = string(rune(id + '0'))
	}
	if resp, ok := m.responses[idStr]; ok {
		return resp, nil
	}
	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]interface{}{},
	}, nil
}

func (m *mockTransport) SendNotification(ctx context.Context, notif Notification) error {
	return nil
}

func (m *mockTransport) ReceiveMessage(ctx context.Context) (interface{}, error) {
	return nil, nil
}

func (m *mockTransport) Close() error {
	m.closed = true
	return nil
}
