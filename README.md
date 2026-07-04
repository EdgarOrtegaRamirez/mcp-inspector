# MCP Inspector

A comprehensive CLI tool for testing, validating, and documenting Model Context Protocol (MCP) server implementations.

[![CI](https://github.com/EdgarOrtegaRamirez/mcp-inspector/actions/workflows/ci.yml/badge.svg)](https://github.com/EdgarOrtegaRamirez/mcp-inspector/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/EdgarOrtegaRamirez/mcp-inspector)](https://goreportcard.com/report/github.com/EdgarOrtegaRamirez/mcp-inspector)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- **Server Discovery** — Connect to MCP servers via stdio or SSE transport
- **Tool Inspection** — List all available tools with their JSON Schema definitions
- **Resource Discovery** — Discover and read resources from the server
- **Prompt Catalog** — List available prompts and their arguments
- **Schema Validation** — Validate tool schemas against JSON Schema standards
- **Integration Testing** — Run automated tests against all discovered tools
- **Documentation Generation** — Generate comprehensive markdown documentation
- **Multiple Output Formats** — Text, JSON, Markdown, and compact output

## Installation

```bash
go install github.com/EdgarOrtegaRamirez/mcp-inspector@latest
```

Or build from source:

```bash
git clone https://github.com/EdgarOrtegaRamirez/mcp-inspector
cd mcp-inspector
go build -o mcp-inspector .
```

## Quick Start

### Inspect a Server

```bash
# Full inspection of an MCP server
mcp-inspector inspect /path/to/mcp-server

# With verbose output
mcp-inspector inspect -v /path/to/mcp-server

# Export as JSON
mcp-inspector inspect -f json /path/to/mcp-server > report.json

# Export as Markdown documentation
mcp-inspector docs -o README.md /path/to/mcp-server
```

### List Tools

```bash
# List all tools with schemas
mcp-inspector tools /path/to/mcp-server

# Verbose mode shows full schema
mcp-inspector tools -v /path/to/mcp-server
```

### Call a Tool

```bash
# Call a tool with JSON arguments
mcp-inspector call /path/to/mcp-server tool-name '{"arg1": "value1"}'
```

### Validate Schemas

```bash
# Validate all tool schemas
mcp-inspector validate /path/to/mcp-server
```

### Run Integration Tests

```bash
# Run tests against all tools
mcp-inspector test /path/to/mcp-server
```

## Output Formats

### Text (default)
Colored terminal output with full details:
```
═══════════════════════════════════════════════════
         MCP Server Inspection Report
═══════════════════════════════════════════════════

Server Information
  Name:     my-mcp-server
  Version:  1.0.0
  Capabilities: tools, resources

Tools (5)
  • read_file ✓ schema
    Read the contents of a file
  • write_file ✓ schema
    Write content to a file
...
```

### JSON
Machine-readable JSON output:
```json
{
  "server_name": "my-mcp-server",
  "server_version": "1.0.0",
  "tools_count": 5,
  "summary": {
    "grade": "A",
    "score": 95
  }
}
```

### Markdown
Generated documentation suitable for README files.

### Compact
One-line summary for CI/CD pipelines:
```
✓ my-mcp-server v1.0.0 | 5 tools, 2 resources, 1 prompts | Score: A (95/100)
```

## Architecture

```
mcp-inspector/
├── cmd/                    # CLI commands
│   └── root.go            # Root command and subcommands
├── pkg/
│   ├── mcp/               # MCP protocol implementation
│   │   ├── types.go       # Protocol types (JSON-RPC 2.0)
│   │   ├── transport.go   # Stdio transport
│   │   ├── sse_transport.go # SSE transport
│   │   └── client.go      # High-level client
│   ├── inspector/         # Core inspection engine
│   │   └── inspector.go   # Inspection logic
│   ├── schema/            # JSON Schema validation
│   │   └── validator.go   # Schema validator
│   └── report/            # Report generation
│       └── report.go      # Multi-format reports
├── main.go                # Entry point
├── go.mod                 # Go module
├── LICENSE                # MIT License
├── README.md              # Documentation
└── .github/
    └── workflows/
        └── ci.yml         # CI pipeline
```

## MCP Protocol

MCP Inspector implements the Model Context Protocol (MCP) 2024-11-05 specification:

- **JSON-RPC 2.0** — All communication uses JSON-RPC 2.0
- **stdio Transport** — Communicate via stdin/stdout pipes
- **SSE Transport** — Communicate via HTTP Server-Sent Events
- **Tool Discovery** — `tools/list` and `tools/call`
- **Resource Access** — `resources/list` and `resources/read`
- **Prompt Catalog** — `prompts/list` and `prompts/get`

## Development

### Prerequisites

- Go 1.24+

### Build

```bash
go build -o mcp-inspector .
```

### Test

```bash
go test ./...
```

### Lint

```bash
golangci-lint run
```

## License

MIT License — see [LICENSE](LICENSE) for details.
