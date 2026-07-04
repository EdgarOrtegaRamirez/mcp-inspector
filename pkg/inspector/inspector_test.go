package inspector

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/EdgarOrtegaRamirez/mcp-inspector/pkg/mcp"
)

// mockTransport for testing
type mockTransport struct {
	responses map[string]*mcp.Response
	closed    bool
}

func (m *mockTransport) SendRequest(ctx context.Context, req mcp.Request) (*mcp.Response, error) {
	if resp, ok := m.responses[req.Method]; ok {
		return resp, nil
	}
	return &mcp.Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]interface{}{},
	}, nil
}

func (m *mockTransport) SendNotification(ctx context.Context, notif mcp.Notification) error {
	return nil
}

func (m *mockTransport) ReceiveMessage(ctx context.Context) (interface{}, error) {
	return nil, nil
}

func (m *mockTransport) Close() error {
	m.closed = true
	return nil
}

func TestInspectorNew(t *testing.T) {
	transport := &mockTransport{}
	client := mcp.NewClient(transport)
	insp := NewInspector(client)
	if insp == nil {
		t.Fatal("inspector is nil")
	}
}

func TestInspectorInitialize(t *testing.T) {
	initResult, _ := json.Marshal(map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "test-server",
			"version": "1.0.0",
		},
	})

	transport := &mockTransport{
		responses: map[string]*mcp.Response{
			"initialize": {
				JSONRPC: "2.0",
				ID:      1,
				Result:  json.RawMessage(initResult),
			},
		},
	}

	client := mcp.NewClient(transport)
	insp := NewInspector(client)

	ctx := context.Background()
	serverCaps, err := client.Initialize(ctx)
	if err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	insp.server = serverCaps

	if serverCaps.Info.Name != "test-server" {
		t.Errorf("expected server name test-server, got %s", serverCaps.Info.Name)
	}
}

func TestToolResult(t *testing.T) {
	result := TestResult{
		Name:     "test-tool",
		Type:     "tool_call",
		Status:   "pass",
		Duration: 100 * time.Millisecond,
	}

	if result.Name != "test-tool" {
		t.Errorf("expected name test-tool, got %s", result.Name)
	}
	if result.Status != "pass" {
		t.Errorf("expected status pass, got %s", result.Status)
	}
}

func TestSummaryCalculation(t *testing.T) {
	insp := &Inspector{
		results: &InspectionResults{
			Summary: &Summary{},
			TestResults: []TestResult{
				{Status: "pass"},
				{Status: "pass"},
				{Status: "fail"},
				{Status: "error"},
			},
		},
	}

	insp.calculateSummary()

	summary := insp.results.Summary
	if summary.TotalTests != 4 {
		t.Errorf("expected 4 total tests, got %d", summary.TotalTests)
	}
	if summary.Passed != 2 {
		t.Errorf("expected 2 passed, got %d", summary.Passed)
	}
	if summary.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", summary.Failed)
	}
	if summary.Errors != 1 {
		t.Errorf("expected 1 error, got %d", summary.Errors)
	}
	if summary.Score != 50 {
		t.Errorf("expected score 50, got %d", summary.Score)
	}
	if summary.Grade != "F" {
		t.Errorf("expected grade F, got %s", summary.Grade)
	}
}

func TestGradeAssignment(t *testing.T) {
	tests := []struct {
		score int
		grade string
	}{
		{100, "A"},
		{95, "A"},
		{90, "A"},
		{85, "B"},
		{80, "B"},
		{75, "C"},
		{70, "C"},
		{65, "D"},
		{60, "D"},
		{50, "F"},
		{0, "F"},
	}

	for _, tt := range tests {
		// Set up TestResults to produce the desired score
		totalTests := 10
		passedTests := tt.score * totalTests / 100
		
		testResults := make([]TestResult, totalTests)
		for i := 0; i < passedTests; i++ {
			testResults[i] = TestResult{Status: "pass"}
		}
		for i := passedTests; i < totalTests; i++ {
			testResults[i] = TestResult{Status: "fail"}
		}
		
		insp := &Inspector{
			results: &InspectionResults{
				Summary:     &Summary{},
				TestResults: testResults,
			},
		}
		insp.calculateSummary()
		if insp.results.Summary.Grade != tt.grade {
			t.Errorf("score %d: expected grade %s, got %s", tt.score, tt.grade, insp.results.Summary.Grade)
		}
	}
}

func TestExportJSON(t *testing.T) {
	insp := &Inspector{
		results: &InspectionResults{
			StartTime: time.Now(),
			EndTime:   time.Now(),
			ServerInfo: &ServerInfoResult{
				Name:    "test",
				Version: "1.0.0",
			},
			Summary: &Summary{
				TotalTests: 0,
				Passed:     0,
				Failed:     0,
				Errors:     0,
				Score:      100,
				Grade:      "A",
			},
		},
	}

	data, err := insp.ExportJSON()
	if err != nil {
		t.Fatalf("failed to export JSON: %v", err)
	}

	var result InspectionResults
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if result.ServerInfo.Name != "test" {
		t.Errorf("expected server name test, got %s", result.ServerInfo.Name)
	}
}
