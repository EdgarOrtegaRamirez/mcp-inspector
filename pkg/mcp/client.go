package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// Client is a high-level MCP client
type Client struct {
	transport Transport
	nextID    int
	server    *ServerCapabilities
}

// ServerCapabilities stores server capabilities discovered during initialization
type ServerCapabilities struct {
	Info         ServerInfo
	Capabilities Capabilities
}

// NewClient creates a new MCP client with the given transport
func NewClient(transport Transport) *Client {
	return &Client{
		transport: transport,
		nextID:    1,
	}
}

func (c *Client) nextRequestID() int {
	id := c.nextID
	c.nextID++
	return id
}

// Initialize performs the MCP initialization handshake
func (c *Client) Initialize(ctx context.Context) (*ServerCapabilities, error) {
	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]string{
			"name":    "mcp-inspector",
			"version": "1.0.0",
		},
	}

	req := NewRequest(c.nextRequestID(), "initialize", params)
	resp, err := c.transport.SendRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("initialize error: %d - %s", resp.Error.Code, resp.Error.Message)
	}

	// Parse result
	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result InitializeResult
	if err := json.Unmarshal(resultData, &result); err != nil {
		return nil, fmt.Errorf("failed to parse initialize result: %w", err)
	}

	c.server = &ServerCapabilities{
		Info:         result.ServerInfo,
		Capabilities: result.Capabilities,
	}

	// Send initialized notification
	notif := Notification{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	if err := c.transport.SendNotification(ctx, notif); err != nil {
		return nil, fmt.Errorf("failed to send initialized notification: %w", err)
	}

	return c.server, nil
}

// ListTools lists available tools from the server
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	req := NewRequest(c.nextRequestID(), "tools/list", nil)
	resp, err := c.transport.SendRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list tools failed: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("list tools error: %d - %s", resp.Error.Code, resp.Error.Message)
	}

	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result ListToolsResult
	if err := json.Unmarshal(resultData, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tools result: %w", err)
	}

	return result.Tools, nil
}

// CallTool calls a tool with the given arguments
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*CallToolResult, error) {
	params := CallToolParams{
		Name:      name,
		Arguments: args,
	}

	req := NewRequest(c.nextRequestID(), "tools/call", params)
	resp, err := c.transport.SendRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("call tool failed: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("call tool error: %d - %s", resp.Error.Code, resp.Error.Message)
	}

	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result CallToolResult
	if err := json.Unmarshal(resultData, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}

	return &result, nil
}

// ListResources lists available resources from the server
func (c *Client) ListResources(ctx context.Context) ([]Resource, error) {
	req := NewRequest(c.nextRequestID(), "resources/list", nil)
	resp, err := c.transport.SendRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list resources failed: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("list resources error: %d - %s", resp.Error.Code, resp.Error.Message)
	}

	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result ListResourcesResult
	if err := json.Unmarshal(resultData, &result); err != nil {
		return nil, fmt.Errorf("failed to parse resources result: %w", err)
	}

	return result.Resources, nil
}

// ReadResource reads a resource by URI
func (c *Client) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	params := ReadResourceParams{URI: uri}

	req := NewRequest(c.nextRequestID(), "resources/read", params)
	resp, err := c.transport.SendRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("read resource failed: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("read resource error: %d - %s", resp.Error.Code, resp.Error.Message)
	}

	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result ReadResourceResult
	if err := json.Unmarshal(resultData, &result); err != nil {
		return nil, fmt.Errorf("failed to parse resource result: %w", err)
	}

	return &result, nil
}

// ListPrompts lists available prompts from the server
func (c *Client) ListPrompts(ctx context.Context) ([]Prompt, error) {
	req := NewRequest(c.nextRequestID(), "prompts/list", nil)
	resp, err := c.transport.SendRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list prompts failed: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("list prompts error: %d - %s", resp.Error.Code, resp.Error.Message)
	}

	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result ListPromptsResult
	if err := json.Unmarshal(resultData, &result); err != nil {
		return nil, fmt.Errorf("failed to parse prompts result: %w", err)
	}

	return result.Prompts, nil
}

// GetPrompt gets a prompt by name with arguments
func (c *Client) GetPrompt(ctx context.Context, name string, args map[string]interface{}) (*GetPromptResult, error) {
	params := GetPromptParams{
		Name:      name,
		Arguments: args,
	}

	req := NewRequest(c.nextRequestID(), "prompts/get", params)
	resp, err := c.transport.SendRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get prompt failed: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("get prompt error: %d - %s", resp.Error.Code, resp.Error.Message)
	}

	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result GetPromptResult
	if err := json.Unmarshal(resultData, &result); err != nil {
		return nil, fmt.Errorf("failed to parse prompt result: %w", err)
	}

	return &result, nil
}

// GetServerCapabilities returns the server capabilities (after Initialize)
func (c *Client) GetServerCapabilities() *ServerCapabilities {
	return c.server
}

// Close closes the client and transport
func (c *Client) Close() error {
	return c.transport.Close()
}
