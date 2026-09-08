# powerctl Architecture

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
│   └── powerctl/
│       └── main.go              # Entry point, command registration
├── internal/
│   ├── api/
│   │   ├── client.go            # GraphQL HTTP client
│   │   ├── client_test.go
│   │   ├── queries.go           # GraphQL query definitions
│   │   ├── websocket.go         # WebSocket for live streaming
│   │   ├── websocket_test.go
│   │   ├── reconnect.go         # Backoff, hourly budget, error taxonomy
│   │   ├── reconnect_test.go
│   │   └── livestream_test.go   # End-to-end against a local WebSocket server
│   ├── commands/
│   │   ├── root.go              # Root command, global flags
│   │   ├── config.go            # `powerctl config` - setup wizard
│   │   ├── home.go              # `powerctl home`
│   │   ├── prices.go            # `powerctl prices`
│   │   ├── consumption.go       # `powerctl consumption`
│   │   ├── live.go              # `powerctl live`
│   │   └── version.go           # `powerctl version`
│   ├── config/
│   │   ├── config.go            # Configuration loading
│   │   └── config_test.go
│   ├── models/
│   │   └── types.go             # Data structures
│   └── output/
│       ├── formatter.go         # Formatter interface
│       ├── formatter_test.go
│       ├── pretty.go            # Beautiful CLI output (default)
│       ├── json.go              # JSON formatter
│       └── markdown.go          # Markdown formatter
├── .goreleaser.yml              # Build, release and Homebrew cask publishing
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── CONTEXT.md                   # Domain vocabulary
├── CONTRIBUTING.md
└── ARCHITECTURE.md
```

## Component Overview

### Entry Point (`cmd/powerctl/main.go`)

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
- Requires `User-Agent: powerctl-cli/1.0` — Tibber rejects the default Go client
- Heartbeat: none. The connection lives as long as the server keeps it open
- Graceful shutdown on SIGINT/SIGTERM, including mid-backoff
- A GraphQL error payload or a null `liveMeasurement` becomes an error rather
  than a nil measurement handed to the formatter (see #13)

#### Reconnection (`reconnect.go`)

`Subscribe` stays a single-attempt primitive; `SubscribeWithReconnect` layers
the retry on top, so the connection code stays testable without a retry loop
running through it.

- Backoff: 5s, doubling to a 5-minute ceiling, with equal jitter (half the delay
  fixed, half random)
- Budget: at most 10 connection attempts per rolling hour. Tibber allows 20, and
  a failing stream must not spend the allowance a manual restart needs. Hitting
  the cap ends the stream with `ErrReconnectBudget`
- Reset: a connection that delivered data and stayed up for a minute returns the
  backoff to 5s
- `--no-reconnect` skips the layer entirely and calls `Subscribe` directly

Whether an error is retried is the substance of it, since a wrong answer either
drops a recoverable stream or burns the hourly budget on a request that can
never succeed:

| Condition | Behaviour |
|-----------|-----------|
| Read error, abnormal or server-initiated close | Retry |
| Handshake refused with HTTP 401/403 | Fatal, exit 1 |
| Close code `4403 Invalid token` | Fatal, exit 1 |
| `connection_error` in place of `connection_ack` | Fatal, exit 1 |
| Unknown home ID, other GraphQL errors (see #13) | Fatal, exit 1 |
| Ack timeout (10s, internal deadline) | Retry |
| `ctx` cancelled by SIGINT/SIGTERM | Clean exit 0 |

Fatal errors are tagged at the point they are raised (`markFatal`) rather than
matched on message text later.

### Commands (`internal/commands/`)

| Command | Input | Output | Exit Codes |
|---------|-------|--------|------------|
| `config init` | interactive | Setup wizard | 0=OK, 1=Error |
| `config show` | - | Current config | 0=OK |
| `config set` | key value | Confirmation | 0=OK, 1=Error |
| `home` | - | Home info | 0=OK, 1=Error |
| `prices` | - | Price list | 0=OK, 1=Error |
| `consumption`| `--resolution`, `--last`, `--home-id` | Consumption history | 0=OK, 1=Error |
| `live` | `--home-id`, `--no-reconnect` | Stream | 0=Clean exit, 1=Error |
| `version` | - | Version, commit, build date | 0=OK |

### Output Formatters (`internal/output/`)

```go
type Formatter interface {
    FormatHome(home *models.HomeResponse) string
    FormatHomes(homes []models.HomeResponse) string
    FormatPrices(prices *models.PriceInfo, homeID string) string
    FormatLiveMeasurement(m *models.LiveMeasurement) string
    FormatConsumptionHistory(nodes []models.ConsumptionNode, resolution string) string
}
```

Adding a method to this interface means implementing it in all three
formatters below — there is no embedded default.

Three implementations:
- `PrettyFormatter` - Beautiful CLI output with colors (default)
- `JSONFormatter` - Indented JSON, except `FormatLiveMeasurement` which is
  compact (one object per line) for streaming
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
| Dropped live stream | Reconnect with bounded backoff; exit 1 once the hourly budget is spent |
| Auth errors | Clear message: "Invalid token" |
| No Pulse | Exit 1 with "Pulse not enabled" |
| Parse errors | Log raw response, exit 1 |

## Cross-Platform Build

`make build-all` fans out to per-OS targets:

```makefile
build-all: build-linux build-darwin build-windows
```

covering linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 and
windows/amd64. Release builds do not use this — see below.

## Dependencies

| Package | Purpose | Why |
|---------|---------|-----|
| `spf13/cobra` | CLI framework | Industry standard (kubectl, hugo) |
| `spf13/viper` | Config loading | Handles env + file + flags |
| `coder/websocket` | WebSocket | Pure Go, well maintained (formerly `nhooyr.io/websocket`) |
| `gopkg.in/yaml.v3` | Config file writing | Used by `config init`/`config set` |

## Testing Strategy

```
internal/
├── api/
│   ├── client_test.go       # Mock HTTP responses
│   ├── websocket_test.go    # Live payload parsing and error mapping
│   ├── reconnect_test.go    # Backoff and budget, on a faked clock
│   └── livestream_test.go   # Reconnect over a real socket, via httptest
├── config/
│   └── config_test.go       # Config resolution order
└── output/
    └── formatter_test.go    # Formatter output assertions
```

`make check` runs the same gates as CI: `gofmt` check, `go vet`, `go mod verify`
and `go test -race ./...`.

Demo token for testing: `5K4MVS-OjfWhK_4yrjOlFe1F6kJXPVf7eQYggo8ebAE`

## Release Pipeline

Tagging `v*.*.*` triggers `.github/workflows/release.yml`, which runs the test
suite and then GoReleaser:

1. Cross-compiles six targets — linux, darwin and windows on amd64 and arm64.
   These are the release binaries, not `make build-all`.
2. Archives them (`tar.gz`, `zip` on windows) and writes `checksums.txt`.
3. Creates the GitHub release, with grouped notes and an install footer built
   from the `changelog` and `release` blocks in `.goreleaser.yml`.
4. Generates `Casks/powerctl.rb` and pushes it to
   [KristofferRisa/homebrew-powerctl](https://github.com/KristofferRisa/homebrew-powerctl),
   authenticated with the `HOMEBREW_TAP_TOKEN` secret.

The tap is generated output — never hand-edit the cask; it is overwritten by
the next release. `replace_existing_artifacts: true` lets a failed release job
be re-run without colliding with assets the first attempt already uploaded.

## Security Considerations

1. Token never logged or printed
2. Config file permissions checked (warn if world-readable)
3. No shell expansion in any path handling
4. WebSocket TLS verification enabled
