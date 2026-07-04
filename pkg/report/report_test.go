package report

import (
	"bytes"
	"testing"
)

func TestTextReport(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewReporter(&buf)

	data := &ReportData{
		ServerName:    "test-server",
		ServerVersion: "1.0.0",
		Capabilities:  []string{"tools", "resources"},
		ToolsCount:    2,
		Tools: []ToolData{
			{Name: "tool1", Description: "First tool", HasSchema: true},
			{Name: "tool2", Description: "Second tool", HasSchema: false},
		},
		ResourcesCount: 1,
		Resources: []ResourceData{
			{URI: "file:///test", Name: "test.txt", Description: "Test file"},
		},
		PromptsCount: 0,
		Summary: SummaryData{
			TotalTests: 10,
			Passed:     8,
			Failed:     1,
			Errors:     1,
			Score:      80,
			Grade:      "B",
		},
	}

	err := reporter.TextReport(data)
	if err != nil {
		t.Fatalf("failed to generate text report: %v", err)
	}

	output := buf.String()
	if !contains(output, "test-server") {
		t.Error("expected server name in output")
	}
	if !contains(output, "tool1") {
		t.Error("expected tool1 in output")
	}
}

func TestJSONReport(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewReporter(&buf)

	data := &ReportData{
		ServerName:   "test-server",
		ServerVersion: "1.0.0",
		Summary: SummaryData{
			Grade: "A",
			Score: 95,
		},
	}

	err := reporter.JSONReport(data)
	if err != nil {
		t.Fatalf("failed to generate JSON report: %v", err)
	}

	output := buf.String()
	if !contains(output, "test-server") {
		t.Error("expected server name in output")
	}
	if !contains(output, "\"Grade\": \"A\"") {
		t.Error("expected grade A in output")
	}
}

func TestMarkdownReport(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewReporter(&buf)

	data := &ReportData{
		ServerName:   "test-server",
		ServerVersion: "1.0.0",
		ToolsCount:   1,
		Tools: []ToolData{
			{Name: "my-tool", Description: "A test tool", HasSchema: true},
		},
		Summary: SummaryData{
			Grade: "A",
			Score: 100,
		},
	}

	err := reporter.MarkdownReport(data)
	if err != nil {
		t.Fatalf("failed to generate markdown report: %v", err)
	}

	output := buf.String()
	if !contains(output, "# MCP Server Inspection Report") {
		t.Error("expected markdown header")
	}
	if !contains(output, "my-tool") {
		t.Error("expected tool name in output")
	}
}

func TestCompactReport(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewReporter(&buf)

	data := &ReportData{
		ServerName:    "test-server",
		ServerVersion: "1.0.0",
		ToolsCount:    3,
		ResourcesCount: 2,
		PromptsCount:  1,
		Summary: SummaryData{
			Grade: "A",
			Score: 95,
		},
	}

	err := reporter.CompactReport(data)
	if err != nil {
		t.Fatalf("failed to generate compact report: %v", err)
	}

	output := buf.String()
	if !contains(output, "test-server") {
		t.Error("expected server name in output")
	}
	if !contains(output, "3 tools") {
		t.Error("expected tool count in output")
	}
}

func TestEmptyReport(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewReporter(&buf)

	data := &ReportData{
		ServerName:    "empty-server",
		ServerVersion: "0.0.1",
		Summary:       SummaryData{Grade: "N/A"},
	}

	err := reporter.TextReport(data)
	if err != nil {
		t.Fatalf("failed to generate text report: %v", err)
	}

	output := buf.String()
	if !contains(output, "empty-server") {
		t.Error("expected server name in output")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
