package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/EdgarOrtegaRamirez/mcp-inspector/pkg/inspector"
	"github.com/EdgarOrtegaRamirez/mcp-inspector/pkg/mcp"
	"github.com/EdgarOrtegaRamirez/mcp-inspector/pkg/report"
	"github.com/EdgarOrtegaRamirez/mcp-inspector/pkg/schema"
)

var (
	flagFormat  string
	flagOutput  string
	flagTimeout int
	flagVerbose bool
)

var rootCmd = &cobra.Command{
	Use:   "mcp-inspector",
	Short: "MCP Server Inspection & Validation Tool",
	Long: `MCP Inspector is a CLI tool for testing, validating, and documenting
Model Context Protocol (MCP) server implementations.

It connects to MCP servers, discovers their capabilities, tests tools/resources/prompts,
validates schemas, and generates comprehensive reports.`,
}

var inspectCmd = &cobra.Command{
	Use:   "inspect [command...]",
	Short: "Inspect an MCP server",
	Long: `Connect to an MCP server and perform a full inspection including
discovery of tools, resources, and prompts, with schema validation.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runInspect,
}

var toolsCmd = &cobra.Command{
	Use:   "tools [command...]",
	Short: "List tools from an MCP server",
	Long:  "Connect to an MCP server and list all available tools with their schemas.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runTools,
}

var callCmd = &cobra.Command{
	Use:   "call [command...] [tool-name] [args-json]",
	Short: "Call a tool on an MCP server",
	Long:  "Call a specific tool on an MCP server with JSON arguments.",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runCall,
}

var validateCmd = &cobra.Command{
	Use:   "validate [command...]",
	Short: "Validate MCP server schema compliance",
	Long: `Validate that an MCP server's tool schemas are valid JSON Schema
and that tool responses match expected formats.`,
	Args: cobra.MinimumNArgs(1),
	RunE:  runValidate,
}

var testCmd = &cobra.Command{
	Use:   "test [command...]",
	Short: "Run integration tests against an MCP server",
	Long: `Run integration tests against an MCP server by calling all tools
with sample data and verifying responses.`,
	Args: cobra.MinimumNArgs(1),
	RunE:  runTest,
}

var docsCmd = &cobra.Command{
	Use:   "docs [command...]",
	Short: "Generate documentation for an MCP server",
	Long:  "Connect to an MCP server and generate comprehensive documentation.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runDocs,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("mcp-inspector v1.0.0\n")
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagFormat, "format", "f", "text", "Output format: text, json, markdown, compact")
	rootCmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "", "Output file (default: stdout)")
	rootCmd.PersistentFlags().IntVarP(&flagTimeout, "timeout", "t", 30, "Timeout in seconds")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Verbose output")

	rootCmd.AddCommand(inspectCmd)
	rootCmd.AddCommand(toolsCmd)
	rootCmd.AddCommand(callCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(docsCmd)
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func connectToServer(command string, args []string) (*mcp.Client, error) {
	transport, err := mcp.NewStdioTransport(command, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	if err := transport.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	client := mcp.NewClient(transport)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flagTimeout)*time.Second)
	defer cancel()

	if _, err := client.Initialize(ctx); err != nil {
		transport.Close()
		return nil, fmt.Errorf("failed to initialize: %w", err)
	}

	return client, nil
}

func runInspect(cmd *cobra.Command, args []string) error {
	command := args[0]
	serverArgs := args[1:]

	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	if flagVerbose {
		fmt.Fprintf(os.Stderr, "%s Connecting to %s %s\n",
			bold("→"), cyan(command), strings.Join(serverArgs, " "))
	}

	client, err := connectToServer(command, serverArgs)
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flagTimeout)*time.Second)
	defer cancel()

	insp := inspector.NewInspector(client)
	results, err := insp.RunInspection(ctx)
	if err != nil {
		return fmt.Errorf("inspection failed: %w", err)
	}

	data := convertToReportData(results)

	var writer *os.File
	if flagOutput != "" {
		writer, err = os.Create(flagOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer writer.Close()
	} else {
		writer = os.Stdout
	}

	reporter := report.NewReporter(writer)

	switch flagFormat {
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

func runTools(cmd *cobra.Command, args []string) error {
	command := args[0]
	serverArgs := args[1:]

	client, err := connectToServer(command, serverArgs)
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flagTimeout)*time.Second)
	defer cancel()

	tools, err := client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	bold := color.New(color.Bold).SprintFunc()

	fmt.Printf("\n%s\n\n", bold("Available Tools"))

	if len(tools) == 0 {
		fmt.Println("  (no tools available)")
		return nil
	}

	for _, tool := range tools {
		schemaStatus := color.GreenString("✓ schema")
		if tool.InputSchema == nil {
			schemaStatus = color.YellowString("✗ no schema")
		}
		fmt.Printf("  %s %s\n", bold(tool.Name), schemaStatus)
		if tool.Description != "" {
			fmt.Printf("    %s\n", tool.Description)
		}
		if tool.InputSchema != nil && flagVerbose {
			schemaJSON, _ := json.MarshalIndent(tool.InputSchema, "    ", "  ")
			fmt.Printf("    Schema: %s\n", schemaJSON)
		}
		fmt.Println()
	}

	return nil
}

func runCall(cmd *cobra.Command, args []string) error {
	command := args[0]
	toolName := args[1]

	var toolArgs map[string]interface{}
	if len(args) > 2 {
		argsJSON := args[2]
		if err := json.Unmarshal([]byte(argsJSON), &toolArgs); err != nil {
			return fmt.Errorf("invalid arguments JSON: %w", err)
		}
	}

	client, err := connectToServer(command, args[2:])
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flagTimeout)*time.Second)
	defer cancel()

	result, err := client.CallTool(ctx, toolName, toolArgs)
	if err != nil {
		return fmt.Errorf("tool call failed: %w", err)
	}

	if result.IsError {
		fmt.Fprintf(os.Stderr, "%s\n", color.RedString("Tool returned error"))
		for _, content := range result.Content {
			fmt.Println(content.Text)
		}
		return fmt.Errorf("tool returned error")
	}

	for _, content := range result.Content {
		fmt.Println(content.Text)
	}

	return nil
}

func runValidate(cmd *cobra.Command, args []string) error {
	command := args[0]
	serverArgs := args[1:]

	client, err := connectToServer(command, serverArgs)
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flagTimeout)*time.Second)
	defer cancel()

	tools, err := client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	bold := color.New(color.Bold).SprintFunc()

	fmt.Printf("\n%s\n\n", bold("Schema Validation"))

	passed := 0
	failed := 0

	for _, tool := range tools {
		if tool.InputSchema == nil {
			fmt.Printf("  %s %s - %s\n", color.YellowString("SKIP"), tool.Name, "no schema")
			continue
		}

		schemaData, err := json.Marshal(tool.InputSchema)
		if err != nil {
			fmt.Printf("  %s %s - %s\n", color.RedString("FAIL"), tool.Name, "invalid schema format")
			failed++
			continue
		}

		validator, err := schema.NewValidator(schemaData)
		if err != nil {
			fmt.Printf("  %s %s - %s\n", color.RedString("FAIL"), tool.Name, err.Error())
			failed++
			continue
		}

		// Validate schema structure itself
		result := validator.Validate(tool.InputSchema)
		if result.Valid {
			fmt.Printf("  %s %s\n", color.GreenString("PASS"), tool.Name)
			passed++
		} else {
			fmt.Printf("  %s %s - %d errors\n", color.RedString("FAIL"), tool.Name, len(result.Errors))
			for _, e := range result.Errors {
				fmt.Printf("    - %s: %s\n", e.Path, e.Message)
			}
			failed++
		}
	}

	fmt.Printf("\n%s\n", bold(fmt.Sprintf("Results: %d passed, %d failed", passed, failed)))

	return nil
}

func runTest(cmd *cobra.Command, args []string) error {
	command := args[0]
	serverArgs := args[1:]

	client, err := connectToServer(command, serverArgs)
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flagTimeout)*time.Second)
	defer cancel()

	insp := inspector.NewInspector(client)
	if err := insp.Initialize(ctx); err != nil {
		return err
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	bold := color.New(color.Bold).SprintFunc()

	fmt.Printf("\n%s\n\n", bold("Running Integration Tests"))

	passed := 0
	failed := 0

	for _, tool := range tools {
		fmt.Printf("  Testing %s... ", bold(tool.Name))

		sampleArgs := generateSampleArgs(tool.InputSchema)

		result := insp.TestTool(ctx, tool.Name, sampleArgs)
		if result.Status == "pass" {
			fmt.Printf("%s\n", color.GreenString("PASS"))
			passed++
		} else if result.Status == "fail" {
			fmt.Printf("%s - %s\n", color.RedString("FAIL"), result.Error)
			failed++
		} else {
			fmt.Printf("%s - %s\n", color.YellowString("ERROR"), result.Error)
			failed++
		}
	}

	fmt.Printf("\n%s\n", bold(fmt.Sprintf("Results: %d passed, %d failed", passed, failed)))

	return nil
}

func runDocs(cmd *cobra.Command, args []string) error {
	command := args[0]
	serverArgs := args[1:]

	client, err := connectToServer(command, serverArgs)
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flagTimeout)*time.Second)
	defer cancel()

	insp := inspector.NewInspector(client)
	results, err := insp.RunInspection(ctx)
	if err != nil {
		return fmt.Errorf("inspection failed: %w", err)
	}

	data := convertToReportData(results)

	var writer *os.File
	if flagOutput != "" {
		writer, err = os.Create(flagOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer writer.Close()
	} else {
		writer = os.Stdout
	}

	reporter := report.NewReporter(writer)
	return reporter.MarkdownReport(data)
}

// Helper functions
func convertToReportData(results *inspector.InspectionResults) *report.ReportData {
	data := &report.ReportData{
		ServerName:    "unknown",
		ServerVersion: "unknown",
		Summary:       report.SummaryData{Grade: "N/A"},
	}

	if results.ServerInfo != nil {
		data.ServerName = results.ServerInfo.Name
		data.ServerVersion = results.ServerInfo.Version
		data.Capabilities = results.ServerInfo.Capabilities
	}

	if results.Tools != nil {
		data.ToolsCount = results.Tools.Count
		data.Tools = make([]report.ToolData, len(results.Tools.Tools))
		for i, t := range results.Tools.Tools {
			data.Tools[i] = report.ToolData{
				Name:        t.Name,
				Description: t.Description,
				HasSchema:   t.HasSchema,
			}
		}
	}

	if results.Resources != nil {
		data.ResourcesCount = results.Resources.Count
		data.Resources = make([]report.ResourceData, len(results.Resources.Resources))
		for i, r := range results.Resources.Resources {
			data.Resources[i] = report.ResourceData{
				URI:         r.URI,
				Name:        r.Name,
				Description: r.Description,
				MimeType:    r.MimeType,
			}
		}
	}

	if results.Prompts != nil {
		data.PromptsCount = results.Prompts.Count
		data.Prompts = make([]report.PromptData, len(results.Prompts.Prompts))
		for i, p := range results.Prompts.Prompts {
			argsList := make([]string, len(p.Arguments))
			for j, a := range p.Arguments {
				argsList[j] = a.Name
			}
			data.Prompts[i] = report.PromptData{
				Name:        p.Name,
				Description: p.Description,
				Arguments:   argsList,
			}
		}
	}

	data.TestResults = make([]report.TestData, len(results.TestResults))
	for i, t := range results.TestResults {
		data.TestResults[i] = report.TestData{
			Name:     t.Name,
			Type:     t.Type,
			Status:   t.Status,
			Duration: t.Duration.String(),
			Error:    t.Error,
		}
	}

	if results.Summary != nil {
		data.Summary = report.SummaryData{
			TotalTests: results.Summary.TotalTests,
			Passed:     results.Summary.Passed,
			Failed:     results.Summary.Failed,
			Errors:     results.Summary.Errors,
			Score:      results.Summary.Score,
			Grade:      results.Summary.Grade,
		}
	}

	return data
}

func generateSampleArgs(s interface{}) map[string]interface{} {
	args := make(map[string]interface{})

	if schemaMap, ok := s.(map[string]interface{}); ok {
		if properties, ok := schemaMap["properties"].(map[string]interface{}); ok {
			for name, propSchema := range properties {
				if propMap, ok := propSchema.(map[string]interface{}); ok {
					switch propMap["type"] {
					case "string":
						args[name] = "test"
					case "number", "integer":
						args[name] = 1
					case "boolean":
						args[name] = true
					case "array":
						args[name] = []interface{}{}
					case "object":
						args[name] = map[string]interface{}{}
					default:
						args[name] = "test"
					}
				}
			}
		}
	}

	return args
}
