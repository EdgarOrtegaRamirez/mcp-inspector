package inspector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/EdgarOrtegaRamirez/mcp-inspector/pkg/mcp"
)

// Inspector is the core inspection engine
type Inspector struct {
	client   *mcp.Client
	server   *mcp.ServerCapabilities
	results  *InspectionResults
}

// InspectionResults stores all inspection results
type InspectionResults struct {
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time"`
	ServerInfo    *ServerInfoResult      `json:"server_info"`
	Tools         *ToolsResult           `json:"tools,omitempty"`
	Resources     *ResourcesResult       `json:"resources,omitempty"`
	Prompts       *PromptsResult         `json:"prompts,omitempty"`
	TestResults   []TestResult           `json:"test_results,omitempty"`
	Summary       *Summary               `json:"summary"`
}

// ServerInfoResult stores server info
type ServerInfoResult struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
}

// ToolsResult stores tools inspection results
type ToolsResult struct {
	Count  int            `json:"count"`
	Tools  []ToolInfo     `json:"tools"`
}

// ToolInfo stores information about a single tool
type ToolInfo struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	HasSchema   bool        `json:"has_schema"`
	Schema      interface{} `json:"schema,omitempty"`
}

// ResourcesResult stores resources inspection results
type ResourcesResult struct {
	Count     int            `json:"count"`
	Resources []ResourceInfo `json:"resources"`
}

// ResourceInfo stores information about a single resource
type ResourceInfo struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mime_type,omitempty"`
}

// PromptsResult stores prompts inspection results
type PromptsResult struct {
	Count   int          `json:"count"`
	Prompts []PromptInfo `json:"prompts"`
}

// PromptInfo stores information about a single prompt
type PromptInfo struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []ArgumentInfo   `json:"arguments,omitempty"`
}

// ArgumentInfo stores information about a prompt argument
type ArgumentInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

// TestResult stores the result of a test
type TestResult struct {
	Name     string        `json:"name"`
	Type     string        `json:"type"`
	Status   string        `json:"status"` // pass, fail, error
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
	Details  interface{}   `json:"details,omitempty"`
}

// Summary stores the inspection summary
type Summary struct {
	TotalTests int    `json:"total_tests"`
	Passed     int    `json:"passed"`
	Failed     int    `json:"failed"`
	Errors     int    `json:"errors"`
	Score      int    `json:"score"` // 0-100
	Grade      string `json:"grade"` // A, B, C, D, F
}

// NewInspector creates a new inspector
func NewInspector(client *mcp.Client) *Inspector {
	return &Inspector{
		client: client,
		results: &InspectionResults{
			StartTime: time.Now(),
			Summary: &Summary{},
		},
	}
}

// Initialize initializes the MCP connection
func (i *Inspector) Initialize(ctx context.Context) error {
	serverCaps, err := i.client.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}
	i.server = serverCaps
	return nil
}

// RunInspection runs a full inspection of the MCP server
func (i *Inspector) RunInspection(ctx context.Context) (*InspectionResults, error) {
	// Initialize
	serverCaps, err := i.client.Initialize(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialization failed: %w", err)
	}
	i.server = serverCaps

	// Record server info
	i.results.ServerInfo = &ServerInfoResult{
		Name:    serverCaps.Info.Name,
		Version: serverCaps.Info.Version,
	}
	i.recordCapabilities(serverCaps.Capabilities)

	// Discover tools
	if serverCaps.Capabilities.Tools != nil {
		i.discoverTools(ctx)
	}

	// Discover resources
	if serverCaps.Capabilities.Resources != nil {
		i.discoverResources(ctx)
	}

	// Discover prompts
	if serverCaps.Capabilities.Prompts != nil {
		i.discoverPrompts(ctx)
	}

	// Calculate summary
	i.results.EndTime = time.Now()
	i.calculateSummary()

	return i.results, nil
}

func (i *Inspector) recordCapabilities(caps mcp.Capabilities) {
	capabilities := []string{}
	if caps.Tools != nil {
		capabilities = append(capabilities, "tools")
	}
	if caps.Resources != nil {
		capabilities = append(capabilities, "resources")
	}
	if caps.Prompts != nil {
		capabilities = append(capabilities, "prompts")
	}
	if caps.Logging != nil {
		capabilities = append(capabilities, "logging")
	}
	i.results.ServerInfo.Capabilities = capabilities
}

func (i *Inspector) discoverTools(ctx context.Context) {
	tools, err := i.client.ListTools(ctx)
	if err != nil {
		i.results.Tools = &ToolsResult{Count: 0}
		return
	}

	toolInfos := make([]ToolInfo, len(tools))
	for idx, tool := range tools {
		hasSchema := tool.InputSchema != nil
		toolInfos[idx] = ToolInfo{
			Name:        tool.Name,
			Description: tool.Description,
			HasSchema:   hasSchema,
			Schema:      tool.InputSchema,
		}
	}

	i.results.Tools = &ToolsResult{
		Count: len(tools),
		Tools: toolInfos,
	}
}

func (i *Inspector) discoverResources(ctx context.Context) {
	resources, err := i.client.ListResources(ctx)
	if err != nil {
		i.results.Resources = &ResourcesResult{Count: 0}
		return
	}

	resourceInfos := make([]ResourceInfo, len(resources))
	for idx, res := range resources {
		resourceInfos[idx] = ResourceInfo{
			URI:         res.URI,
			Name:        res.Name,
			Description: res.Description,
			MimeType:    res.MimeType,
		}
	}

	i.results.Resources = &ResourcesResult{
		Count:     len(resources),
		Resources: resourceInfos,
	}
}

func (i *Inspector) discoverPrompts(ctx context.Context) {
	prompts, err := i.client.ListPrompts(ctx)
	if err != nil {
		i.results.Prompts = &PromptsResult{Count: 0}
		return
	}

	promptInfos := make([]PromptInfo, len(prompts))
	for idx, prompt := range prompts {
		argInfos := make([]ArgumentInfo, len(prompt.Arguments))
		for argIdx, arg := range prompt.Arguments {
			argInfos[argIdx] = ArgumentInfo{
				Name:        arg.Name,
				Description: arg.Description,
				Required:    arg.Required,
			}
		}
		promptInfos[idx] = PromptInfo{
			Name:        prompt.Name,
			Description: prompt.Description,
			Arguments:   argInfos,
		}
	}

	i.results.Prompts = &PromptsResult{
		Count:   len(prompts),
		Prompts: promptInfos,
	}
}

// TestTool tests a specific tool with given arguments
func (i *Inspector) TestTool(ctx context.Context, name string, args map[string]interface{}) TestResult {
	start := time.Now()
	result := TestResult{
		Name: fmt.Sprintf("call_tool_%s", name),
		Type: "tool_call",
	}

	callResult, err := i.client.CallTool(ctx, name, args)
	result.Duration = time.Since(start)

	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
	} else if callResult.IsError {
		result.Status = "fail"
		result.Error = "tool returned error"
		result.Details = callResult.Content
	} else {
		result.Status = "pass"
		result.Details = callResult.Content
	}

	i.results.TestResults = append(i.results.TestResults, result)
	return result
}

// TestResource tests reading a resource
func (i *Inspector) TestResource(ctx context.Context, uri string) TestResult {
	start := time.Now()
	result := TestResult{
		Name: fmt.Sprintf("read_resource_%s", uri),
		Type: "resource_read",
	}

	readResult, err := i.client.ReadResource(ctx, uri)
	result.Duration = time.Since(start)

	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
	} else {
		result.Status = "pass"
		result.Details = readResult.Contents
	}

	i.results.TestResults = append(i.results.TestResults, result)
	return result
}

// TestPrompt tests getting a prompt
func (i *Inspector) TestPrompt(ctx context.Context, name string, args map[string]interface{}) TestResult {
	start := time.Now()
	result := TestResult{
		Name: fmt.Sprintf("get_prompt_%s", name),
		Type: "prompt_get",
	}

	promptResult, err := i.client.GetPrompt(ctx, name, args)
	result.Duration = time.Since(start)

	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
	} else {
		result.Status = "pass"
		result.Details = promptResult
	}

	i.results.TestResults = append(i.results.TestResults, result)
	return result
}

func (i *Inspector) calculateSummary() {
	summary := i.results.Summary
	summary.TotalTests = len(i.results.TestResults)

	for _, test := range i.results.TestResults {
		switch test.Status {
		case "pass":
			summary.Passed++
		case "fail":
			summary.Failed++
		case "error":
			summary.Errors++
		}
	}

	// Calculate score
	if summary.TotalTests > 0 {
		summary.Score = (summary.Passed * 100) / summary.TotalTests
	} else {
		summary.Score = 100 // No tests = pass
	}

	// Assign grade
	switch {
	case summary.Score >= 90:
		summary.Grade = "A"
	case summary.Score >= 80:
		summary.Grade = "B"
	case summary.Score >= 70:
		summary.Grade = "C"
	case summary.Score >= 60:
		summary.Grade = "D"
	default:
		summary.Grade = "F"
	}
}

// GetResults returns the current results
func (i *Inspector) GetResults() *InspectionResults {
	return i.results
}

// ExportJSON exports results as JSON
func (i *Inspector) ExportJSON() ([]byte, error) {
	return json.MarshalIndent(i.results, "", "  ")
}
