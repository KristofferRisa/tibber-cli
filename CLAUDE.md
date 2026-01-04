# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
make build          # Build binary to ./tibber
make test           # Run all tests
make fmt            # Format code
make lint           # Run golangci-lint
make tidy           # Tidy go.mod
make build-all      # Cross-compile for linux/darwin/windows
make clean          # Remove build artifacts
```

Run a single test:
```bash
go test -v ./internal/config -run TestLoad_EnvVarTakesPriority
```

## Architecture

This is a Go CLI for Tibber power consumption data using Cobra for commands.

**Key flow:** Commands (`internal/commands/`) → API client (`internal/api/`) → Output formatter (`internal/output/`)

**Configuration priority:** CLI flags > `TIBBER_TOKEN` env var > `~/.tibber/config.yaml`

**Output formats:** `pretty` (default, colored), `json`, `markdown` — selected via `--format` flag

**API:** GraphQL at `api.tibber.com`, WebSocket at `websocket-api.tibber.com` for live streaming

## Key Patterns

- Commands use shared `cfg` and `formatter` from `root.go`
- All formatters implement the `Formatter` interface in `output/formatter.go`
- WebSocket requires `User-Agent: tibber-cli/1.0` header (Tibber rejects default Go client)
- Exit code 1 for errors, graceful shutdown on SIGINT/SIGTERM for live streaming
