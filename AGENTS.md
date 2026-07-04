# AGENTS.md

## Overview

MCP Inspector is a CLI tool for testing, validating, and documenting Model Context Protocol (MCP) server implementations.

## Architecture

- **cmd/** — CLI commands using Cobra framework
- **pkg/mcp/** — MCP protocol types, transports (stdio, SSE), and high-level client
- **pkg/inspector/** — Core inspection engine that orchestrates discovery and testing
- **pkg/schema/** — JSON Schema validation and inference
- **pkg/report/** — Multi-format report generation (text, JSON, markdown, compact)

## Key Concepts

1. **Transport Layer** — Abstracts communication with MCP servers (stdio pipes, HTTP SSE)
2. **Client Layer** — High-level API for MCP protocol operations
3. **Inspector** — Orchestrates server discovery, tool testing, and schema validation
4. **Reporter** — Generates human-readable and machine-readable output

## Development Guidelines

- All MCP protocol types are in `pkg/mcp/types.go`
- Transport implementations handle JSON-RPC 2.0 framing
- Tests use mock transports for isolation
- Report generation supports text, JSON, markdown, and compact formats

## Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test ./pkg/mcp/...
go test ./pkg/schema/...
```

## Building

```bash
# Build binary
go build -o mcp-inspector .

# Install globally
go install .
```
