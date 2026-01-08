# Tibber CLI Architecture

A cross-platform CLI tool for Tibber power consumption data, built following Unix philosophy.

## Design Principles

1. **Do one thing well** - Each command has a single responsibility
2. **Composable output** - JSON for piping, Markdown for humans/AI
3. **Fail fast, fail loud** - Clear error messages, non-zero exit codes
4. **Zero configuration to start** - Works with just `TIBBER_TOKEN` env var

## Directory Structure

```
powerctl-cli/
├── cmd/
│   └── tibber/
│       └── main.go              # Entry point, command registration
├── internal/
│   ├── api/
│   │   ├── client.go            # GraphQL HTTP client
│   │   ├── queries.go           # GraphQL query definitions
│   │   └── websocket.go         # WebSocket for live streaming
│   ├── commands/
│   │   ├── root.go              # Root command, global flags
│   │   ├── config.go            # `tibber config` - setup wizard
│   │   ├── home.go              # `tibber home`
│   │   ├── prices.go            # `tibber prices`
│   │   └── live.go              # `tibber live`
│   ├── config/
│   │   └── config.go            # Configuration loading
│   ├── models/
│   │   └── types.go             # Data structures
│   └── output/
│       ├── formatter.go         # Formatter interface
│       ├── pretty.go            # Beautiful CLI output (default)
│       ├── json.go              # JSON formatter
│       └── markdown.go          # Markdown formatter
├── go.mod
├── go.sum
├── Makefile
└── ARCHITECTURE.md
```

## Component Overview

### Entry Point (`cmd/tibber/main.go`)

Minimal entry point following Go conventions:

```go
func main() {
    if err := commands.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### Configuration (`internal/config/`)

Configuration resolution order (first wins):

1. Command-line flags
2. `TIBBER_TOKEN` environment variable
3. Config file (`~/.tibber/config.yaml`)

```yaml
# ~/.tibber/config.yaml
token: "your-api-token"
home_id: "optional-home-id"      # Skip home selection
format: "markdown"               # Default output format
```

### API Layer (`internal/api/`)

#### GraphQL Client (`client.go`)

- Single HTTP client instance (connection pooling)
- Timeout: 30 seconds
- Retry: None (fail fast)
- Auth: Bearer token header

#### WebSocket Client (`websocket.go`)

- Protocol: `graphql-transport-ws`
- Reconnection: Exponential backoff (max 5 retries)
- Heartbeat: 30-second ping interval
- Graceful shutdown on SIGINT/SIGTERM

### Commands (`internal/commands/`)

| Command | Input | Output | Exit Codes |
|---------|-------|--------|------------|
| `config init` | interactive | Setup wizard | 0=OK, 1=Error |
| `config show` | - | Current config | 0=OK |
| `config set` | key value | Confirmation | 0=OK, 1=Error |
| `home` | - | Home info | 0=OK, 1=Error |
| `prices` | - | Price list | 0=OK, 1=Error |
| `live` | `--home-id` | Stream | 0=Clean exit, 1=Error |

### Output Formatters (`internal/output/`)

```go
type Formatter interface {
    FormatHome(home *models.Home) string
    FormatPrices(prices []models.Price) string
    FormatLiveMeasurement(m *models.LiveMeasurement) string
}
```

Three implementations:
- `PrettyFormatter` - Beautiful CLI output with colors (default)
- `JSONFormatter` - Compact JSON, one object per line for streaming
- `MarkdownFormatter` - Tables and headers, AI-readable

## Data Flow

```
┌─────────────┐     ┌──────────────┐     ┌───────────────┐
│   CLI       │────▶│   Command    │────▶│   API Client  │
│   Input     │     │   Handler    │     │   (GraphQL)   │
└─────────────┘     └──────────────┘     └───────────────┘
                           │                     │
                           ▼                     ▼
                    ┌──────────────┐     ┌───────────────┐
                    │  Formatter   │◀────│   Response    │
                    │  (JSON/MD)   │     │   Parser      │
                    └──────────────┘     └───────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │   stdout     │
                    └──────────────┘
```

## API Integration

### Endpoints

| Type | URL |
|------|-----|
| GraphQL | `https://api.tibber.com/v1-beta/gql` |
| WebSocket | `wss://websocket-api.tibber.com/v1-beta/gql/subscriptions` |

### Authentication

All requests include:
```
Authorization: Bearer <token>
```

### Rate Limits

- GraphQL: Standard rate limiting
- WebSocket: 20 connections per hour (Tibber-imposed)

## Error Handling

| Category | Strategy |
|----------|----------|
| Network errors | Log and exit with code 1 |
| Auth errors | Clear message: "Invalid token" |
| No Pulse | Exit 2 with "Pulse not enabled" |
| Parse errors | Log raw response, exit 1 |

## Cross-Platform Build

```makefile
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

build-all:
    @for platform in $(PLATFORMS); do \
        GOOS=$${platform%/*} GOARCH=$${platform#*/} \
        go build -o dist/tibber-$${platform%/*}-$${platform#*/} ./cmd/tibber; \
    done
```

## Dependencies

| Package | Purpose | Why |
|---------|---------|-----|
| `spf13/cobra` | CLI framework | Industry standard (kubectl, hugo) |
| `spf13/viper` | Config loading | Handles env + file + flags |
| `nhooyr/websocket` | WebSocket | Pure Go, well maintained |

## Testing Strategy

```
internal/
├── api/
│   └── client_test.go       # Mock HTTP responses
├── commands/
│   └── prices_test.go       # Integration tests
└── output/
    └── formatter_test.go    # Golden file tests
```

Demo token for testing: `5K4MVS-OjfWhK_4yrjOlFe1F6kJXPVf7eQYggo8ebAE`

## Security Considerations

1. Token never logged or printed
2. Config file permissions checked (warn if world-readable)
3. No shell expansion in any path handling
4. WebSocket TLS verification enabled
