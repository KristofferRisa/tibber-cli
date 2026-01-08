# Tibber CLI

A beautiful command-line tool for monitoring your Tibber power consumption and electricity prices.

<p align="center">
  <img src="https://img.shields.io/github/v/release/kristofferrisa/powerctl-cli" alt="Release">
  <img src="https://img.shields.io/github/actions/workflow/status/kristofferrisa/powerctl-cli/test.yml" alt="Tests">
  <img src="https://img.shields.io/github/license/kristofferrisa/powerctl-cli" alt="License">
</p>

## Features

- ⚡ **Real-time monitoring** - Stream live power consumption from your Tibber Pulse
- 💰 **Price tracking** - View current, today's, and tomorrow's electricity prices
- 🏠 **Home management** - List and manage multiple Tibber homes
- 🎨 **Beautiful output** - Colored, formatted CLI output (or JSON/Markdown)
- 🚀 **Cross-platform** - Works on Linux, macOS, and Windows

## Installation

### Download Binary

Download the latest release for your platform from [Releases](https://github.com/kristofferrisa/powerctl-cli/releases).

### Build from Source

```bash
git clone https://github.com/kristofferrisa/powerctl-cli.git
cd powerctl-cli
make build
./tibber --help
```

## Quick Start

1. **Get your API token** from [developer.tibber.com/settings/access-token](https://developer.tibber.com/settings/access-token)

2. **Run setup wizard:**
   ```bash
   tibber config init
   ```

3. **View your home:**
   ```bash
   tibber home
   ```

## Usage

### Configuration

**Option 1: Environment variable (recommended)**
```bash
export TIBBER_TOKEN="your-token-here"
tibber home
```

**Option 2: Config file**
```bash
tibber config init  # Interactive setup
# or manually edit ~/.tibber/config.yaml
```

**Option 3: Command flag**
```bash
tibber --config /path/to/config.yaml home
```

### Commands

#### View Home Information
```bash
tibber home
```
```
⚡ My House
──────────

  📍 Address
     123 Main Street
     12345 Oslo, Norway

  🏠 Details
     Size:      150 m²
     Residents: 2
     Main Fuse: 25 A

  ⚡ Pulse
     Status: ● Connected
```

#### Check Electricity Prices
```bash
tibber prices
```
```
⚡ Electricity Prices
──────────────────────

  NOW  0.45 NOK/kWh  ● Normal

  📅 Today
   ▶ 14:00 ████████░░░░░░░░░░░░ 0.45 NOK
     15:00 ██████████████░░░░░░ 0.62 NOK
     16:00 ████████████████████ 0.78 NOK
```

#### Stream Live Power Consumption
```bash
tibber live
```
```
⚡ Live Power
──────────────

  1,234 W

  📊 Today
     Consumed: 12.50 kWh
     Cost:     45.30 NOK

  🔌 Grid
     Voltage: 230 / 231 / 229 V
     Current: 5.2 / 3.1 / 4.5 A
```

Press `Ctrl+C` to stop streaming.

### Output Formats

Default output is beautiful colored CLI. Change format with `--format`:

**JSON** (for scripting/piping):
```bash
tibber prices --format json | jq '.current.total'
```

**Markdown** (for AI/documentation):
```bash
tibber home --format markdown
```

## Configuration File

Location: `~/.tibber/config.yaml`

```yaml
token: "your-api-token"
home_id: "optional-default-home-id"  # Skip home selection
format: "pretty"                      # Options: pretty, json, markdown
```

View current config:
```bash
tibber config show
```

Update a value:
```bash
tibber config set format json
```

## Development

### Build
```bash
make build          # Build ./tibber
make build-all      # Cross-compile all platforms
```

### Test
```bash
make test           # Run all tests
go test ./internal/config -run TestLoad  # Run specific test
```

### Lint & Format
```bash
make fmt            # Format code
make lint           # Run linter (requires golangci-lint)
```

## API Information

- **GraphQL endpoint:** `https://api.tibber.com/v1-beta/gql`
- **WebSocket (live):** `wss://websocket-api.tibber.com/v1-beta/gql/subscriptions`
- **Rate limits:** 20 WebSocket connections per hour
- **Documentation:** [developer.tibber.com](https://developer.tibber.com/docs)

## Troubleshooting

**"No API token found"**
- Set `TIBBER_TOKEN` environment variable or run `tibber config init`

**"Pulse not enabled"**
- Ensure your Tibber Pulse is connected and active
- Check status at [tibber.com](https://tibber.com)

**Live stream disconnects**
- Rate limit is 20 connections/hour
- WebSocket auto-reconnects on temporary failures

## Contributing

Contributions welcome! Please read [ARCHITECTURE.md](ARCHITECTURE.md) for code structure details.

1. Fork the repository
2. Create a feature branch (`git checkout -b feat-amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feat-amazing-feature`)
5. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) CLI framework
- Uses Tibber's official GraphQL API
- Inspired by Unix philosophy: do one thing well

---

Made with ⚡ by [Kristoffer Risa](https://github.com/kristofferrisa)
