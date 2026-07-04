# Security Policy

## Reporting Vulnerabilities

If you discover a security vulnerability in MCP Inspector, please report it responsibly:

1. **Do NOT** open a public GitHub issue for security vulnerabilities
2. Email security reports to the maintainers
3. Include steps to reproduce the vulnerability
4. Allow reasonable time for a fix before public disclosure

## Security Considerations

### Process Execution
MCP Inspector launches MCP server processes via stdin/stdout. Ensure:
- Only run servers from trusted sources
- Review server configurations before execution
- Use `--timeout` to prevent hanging processes

### Network Communication
When using SSE transport:
- Connections are made to the specified endpoint
- No authentication is implemented (use network-level controls)
- Consider using VPN or private networks for sensitive servers

### Input Validation
- Tool arguments are validated against JSON Schema when available
- Server responses are parsed and may contain arbitrary content
- Sanitize output when displaying to terminals

## Dependencies

MCP Inspector uses minimal dependencies:
- `github.com/fatih/color` — Terminal colors
- `github.com/spf13/cobra` — CLI framework
- `github.com/xeipuuv/gojsonschema` — JSON Schema validation

Run `go mod verify` to verify dependency checksums.

## Best Practices

1. Run with minimal privileges
2. Use `--timeout` to prevent resource exhaustion
3. Review tool schemas before calling unknown tools
4. Use `--format json` for machine-readable output in pipelines
