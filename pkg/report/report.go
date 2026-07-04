package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/fatih/color"
)

// Reporter generates inspection reports in various formats
type Reporter struct {
	writer io.Writer
}

// NewReporter creates a new reporter
func NewReporter(w io.Writer) *Reporter {
	return &Reporter{writer: w}
}

// ReportData contains all data needed for report generation
type ReportData struct {
	ServerName    string
	ServerVersion string
	Capabilities  []string
	ToolsCount    int
	Tools         []ToolData
	ResourcesCount int
	Resources     []ResourceData
	PromptsCount  int
	Prompts       []PromptData
	TestResults   []TestData
	Summary       SummaryData
}

// ToolData for report
type ToolData struct {
	Name        string
	Description string
	HasSchema   bool
}

// ResourceData for report
type ResourceData struct {
	URI         string
	Name        string
	Description string
	MimeType    string
}

// PromptData for report
type PromptData struct {
	Name        string
	Description string
	Arguments   []string
}

// TestData for report
type TestData struct {
	Name     string
	Type     string
	Status   string
	Duration string
	Error    string
}

// SummaryData for report
type SummaryData struct {
	TotalTests int
	Passed     int
	Failed     int
	Errors     int
	Score      int
	Grade      string
}

// TextReport generates a plain text report
func (r *Reporter) TextReport(data *ReportData) error {
	bold := color.New(color.Bold).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Fprintf(r.writer, "\n%s\n", bold("═══════════════════════════════════════════════════"))
	fmt.Fprintf(r.writer, "%s\n", bold("         MCP Server Inspection Report"))
	fmt.Fprintf(r.writer, "%s\n\n", bold("═══════════════════════════════════════════════════"))

	// Server Info
	fmt.Fprintf(r.writer, "%s\n", bold("Server Information"))
	fmt.Fprintf(r.writer, "  Name:     %s\n", cyan(data.ServerName))
	fmt.Fprintf(r.writer, "  Version:  %s\n", data.ServerVersion)
	fmt.Fprintf(r.writer, "  Capabilities: %s\n\n", strings.Join(data.Capabilities, ", "))

	// Tools
	fmt.Fprintf(r.writer, "%s\n", bold(fmt.Sprintf("Tools (%d)", data.ToolsCount)))
	if data.ToolsCount > 0 {
		for _, tool := range data.Tools {
			schemaStatus := green("✓ schema")
			if !tool.HasSchema {
				schemaStatus = yellow("✗ no schema")
			}
			fmt.Fprintf(r.writer, "  • %s %s\n    %s\n", bold(tool.Name), schemaStatus, tool.Description)
		}
	} else {
		fmt.Fprintf(r.writer, "  (none)\n")
	}
	fmt.Fprintln(r.writer)

	// Resources
	fmt.Fprintf(r.writer, "%s\n", bold(fmt.Sprintf("Resources (%d)", data.ResourcesCount)))
	if data.ResourcesCount > 0 {
		for _, res := range data.Resources {
			fmt.Fprintf(r.writer, "  • %s (%s)\n    %s\n", bold(res.Name), res.URI, res.Description)
		}
	} else {
		fmt.Fprintf(r.writer, "  (none)\n")
	}
	fmt.Fprintln(r.writer)

	// Prompts
	fmt.Fprintf(r.writer, "%s\n", bold(fmt.Sprintf("Prompts (%d)", data.PromptsCount)))
	if data.PromptsCount > 0 {
		for _, p := range data.Prompts {
			args := ""
			if len(p.Arguments) > 0 {
				args = " [" + strings.Join(p.Arguments, ", ") + "]"
			}
			fmt.Fprintf(r.writer, "  • %s%s\n    %s\n", bold(p.Name), args, p.Description)
		}
	} else {
		fmt.Fprintf(r.writer, "  (none)\n")
	}
	fmt.Fprintln(r.writer)

	// Test Results
	if len(data.TestResults) > 0 {
		fmt.Fprintf(r.writer, "%s\n", bold(fmt.Sprintf("Test Results (%d)", len(data.TestResults))))
		for _, test := range data.TestResults {
			status := green("PASS")
			if test.Status == "fail" {
				status = red("FAIL")
			} else if test.Status == "error" {
				status = red("ERROR")
			}
			fmt.Fprintf(r.writer, "  [%s] %s (%s)\n", status, test.Name, test.Duration)
			if test.Error != "" {
				fmt.Fprintf(r.writer, "        %s\n", red(test.Error))
			}
		}
		fmt.Fprintln(r.writer)
	}

	// Summary
	fmt.Fprintf(r.writer, "%s\n", bold("Summary"))
	fmt.Fprintf(r.writer, "  Grade:   %s\n", bold(data.Summary.Grade))
	fmt.Fprintf(r.writer, "  Score:   %d/100\n", data.Summary.Score)
	fmt.Fprintf(r.writer, "  Tests:   %d passed, %d failed, %d errors\n",
		data.Summary.Passed, data.Summary.Failed, data.Summary.Errors)
	fmt.Fprintf(r.writer, "\n%s\n", bold("═══════════════════════════════════════════════════"))

	return nil
}

// JSONReport generates a JSON report
func (r *Reporter) JSONReport(data *ReportData) error {
	encoder := json.NewEncoder(r.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// MarkdownReport generates a Markdown report
func (r *Reporter) MarkdownReport(data *ReportData) error {
	tmpl := `# MCP Server Inspection Report

## Server Information

| Field | Value |
|-------|-------|
| Name | {{.ServerName}} |
| Version | {{.ServerVersion}} |
| Capabilities | {{join .Capabilities}} |

## Tools ({{.ToolsCount}})
{{range .Tools}}
- **{{.Name}}** {{if .HasSchema}}✓ schema{{else}}✗ no schema{{end}}
  {{.Description}}
{{end}}
{{if eq .ToolsCount 0}}(none){{end}}

## Resources ({{.ResourcesCount}})
{{range .Resources}}
- **{{.Name}}** ({{.URI}})
  {{.Description}}
{{end}}
{{if eq .ResourcesCount 0}}(none){{end}}

## Prompts ({{.PromptsCount}})
{{range .Prompts}}
- **{{.Name}}**{{if .Arguments}} [{{join .Arguments}}]{{end}}
  {{.Description}}
{{end}}
{{if eq .PromptsCount 0}}(none){{end}}

## Test Results ({{len .TestResults}})
{{range .TestResults}}
- [{{.Status}}] {{.Name}} ({{.Duration}})
{{if .Error}}  Error: {{.Error}}{{end}}
{{end}}

## Summary

| Metric | Value |
|--------|-------|
| Grade | **{{.Summary.Grade}}** |
| Score | {{.Summary.Score}}/100 |
| Passed | {{.Summary.Passed}} |
| Failed | {{.Summary.Failed}} |
| Errors | {{.Summary.Errors}} |
`

	funcMap := template.FuncMap{
		"join": func(items []string) string {
			return strings.Join(items, ", ")
		},
	}

	t, err := template.New("report").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return err
	}

	return t.Execute(r.writer, data)
}

// CompactReport generates a one-line compact report
func (r *Reporter) CompactReport(data *ReportData) error {
	bold := color.New(color.Bold).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	statusIcon := green("✓")
	if data.Summary.Score < 80 {
		statusIcon = yellow(fmt.Sprintf("%d%%", data.Summary.Score))
	}
	if data.Summary.Score < 60 {
		statusIcon = red("✗")
	}

	fmt.Fprintf(r.writer, "%s %s v%s | %d tools, %d resources, %d prompts | Score: %s (%d/100)\n",
		statusIcon,
		bold(data.ServerName),
		data.ServerVersion,
		data.ToolsCount,
		data.ResourcesCount,
		data.PromptsCount,
		data.Summary.Grade,
		data.Summary.Score,
	)

	return nil
}

var yellow = color.New(color.FgYellow).SprintFunc()

// GenerateReport converts raw results to report data and generates the report
func GenerateReport(results interface{}, format string, writer io.Writer) error {
	// This is a simplified version - in production, you'd properly type-assert
	data := &ReportData{
		ServerName:   "unknown",
		ServerVersion: "unknown",
		Summary:      SummaryData{Grade: "N/A"},
	}

	reporter := NewReporter(writer)

	switch format {
	case "json":
		return reporter.JSONReport(data)
	case "markdown", "md":
		return reporter.MarkdownReport(data)
	case "compact":
		return reporter.CompactReport(data)
	default:
		return reporter.TextReport(data)
	}
}
