# powerctl

A beautiful command-line tool for monitoring your Tibber power consumption and electricity prices.

<p align="center">
  <a href="https://github.com/kristofferrisa/powerctl-cli/releases/latest"><img src="https://img.shields.io/github/v/release/kristofferrisa/powerctl-cli?label=release" alt="Latest release"></a>
  <a href="https://github.com/kristofferrisa/powerctl-cli/releases"><img src="https://img.shields.io/github/release-date/kristofferrisa/powerctl-cli?label=release%20notes" alt="Release notes"></a>
  <a href="https://github.com/kristofferrisa/powerctl-cli/actions/workflows/test.yml"><img src="https://img.shields.io/github/actions/workflow/status/kristofferrisa/powerctl-cli/test.yml?branch=main&amp;label=tests" alt="Tests"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/kristofferrisa/powerctl-cli" alt="License"></a>
</p>

## Features

- ⚡ **Real-time monitoring** - Stream live power consumption from your Tibber Pulse
- 💰 **Price tracking** - View current, today's, and tomorrow's electricity prices
- 🏠 **Home management** - List and manage multiple Tibber homes
- 🎨 **Beautiful output** - Colored, formatted CLI output (or JSON/Markdown)
- 🚀 **Cross-platform** - Works on Linux, macOS, and Windows

## Installation

### Homebrew (macOS & Linux)

```bash
brew install kristofferrisa/powerctl/powerctl
```

Upgrade with `brew upgrade powerctl`.

> Installed the older `powerctl-cli` formula? Replace it with
> `brew uninstall powerctl-cli && brew install kristofferrisa/powerctl/powerctl`.

### Download Binary

Download the latest release for your platform from [Releases](https://github.com/kristofferrisa/powerctl-cli/releases).

### Build from Source

```bash
git clone https://github.com/kristofferrisa/powerctl-cli.git
cd powerctl-cli
make build
./powerctl --help
```

## Quick Start

1. **Get your API token** from [developer.tibber.com/settings/access-token](https://developer.tibber.com/settings/access-token)

2. **Run setup wizard:**
   ```bash
   powerctl config init
   ```

3. **View your home:**
   ```bash
   powerctl home
   ```

## Usage

### Configuration

**Option 1: Environment variable (recommended)**
```bash
export TIBBER_TOKEN="your-token-here"
powerctl home
```

**Option 2: Config file**
```bash
powerctl config init  # Interactive setup
# or manually edit ~/.tibber/config.yaml
```

**Option 3: Command flag**
```bash
powerctl --config /path/to/config.yaml home
```

### Commands

#### Check Version
```bash
powerctl version
```
```
powerctl-cli version 0.4.2
Git commit: a1b2c3d
Built: 2026-08-28T15:30:00Z
```

#### View Home Information
```bash
powerctl home
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
powerctl prices
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

#### View Consumption History
```bash
powerctl consumption --resolution daily --last 7
```
```

📊 Consumption History
────────────────────────

  📅 Period    ⚡ Consumption             💰 Total Cost             📊 Avg Price
  ──────────────────────────────────────────────────────────────────────────────
  2023-10-01   ████████████ 24.50 kWh     ████████████ 120.40 NOK    4.90 NOK/kWh
  2023-10-02   ███████░░░░░ 15.00 kWh     █████░░░░░░░  60.00 NOK    4.00 NOK/kWh
  ──────────────────────────────────────────────────────────────────────────────
  Totals       39.50 kWh                  180.40 NOK
```

#### Stream Live Power Consumption
```bash
powerctl live
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

Monitor a specific home with `--home-id` (otherwise the home from your config
is used, or the first one on the account):
```bash
powerctl live --home-id 96a14971-525a-4420-aae9-e5aefaf46a81
```

`powerctl consumption` takes the same flag.

A dropped connection is retried automatically, so the stream can be left
running. The wait starts at 5 seconds and doubles to a 5-minute ceiling, and no
more than 10 connections are opened in any rolling hour — Tibber allows 20, and
a manual restart needs some left. Progress notes go to stderr, so `--format
json` output stays pipeable.

Failures a reconnect cannot fix — an invalid token, an unknown home ID — still
exit 1 immediately instead of retrying. To get that for every error:

```bash
powerctl live --no-reconnect
```

### Output Formats

Default output is beautiful colored CLI. Change format with `--format`:

**JSON** (for scripting/piping):
```bash
powerctl prices --format json | jq '.current.total'
```

**Markdown** (for AI/documentation):
```bash
powerctl home --format markdown
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
powerctl config show
```

Update a value:
```bash
powerctl config set format json
```

## Development

### Build
```bash
make build          # Build ./powerctl
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
- Set `TIBBER_TOKEN` environment variable or run `powerctl config init`

**"Pulse not enabled"**
- Ensure your Tibber Pulse is connected and active
- Check status at [tibber.com](https://tibber.com)

**"invalid or non-existing home ID"**
- The ID passed to `--home-id` is not on your account. List valid IDs with `powerctl home`

**Live stream disconnects**
- Dropped connections are retried automatically — watch stderr for
  `Reconnecting in ...`
- Rate limit is 20 connections/hour. `powerctl` spends at most 10 of them, then
  stops with "reconnect budget exhausted" rather than taking the rest; wait for
  the hour to roll over before restarting
- `--no-reconnect` restores the old behaviour of exiting 1 on the first error

## Contributing

Contributions welcome — including AI-assisted ones. Please read
[CONTRIBUTING.md](CONTRIBUTING.md) for the workflow and
[ARCHITECTURE.md](ARCHITECTURE.md) for code structure details.

1. Open an issue describing the **Goal**, **Plan** and **Tasks** (the
   [task template](.github/ISSUE_TEMPLATE/task.md) has the structure)
2. Fork the repository
3. Create a feature branch (`git checkout -b feat-amazing-feature`)
4. Commit changes (`git commit -m 'Add amazing feature'`)
5. Push to branch (`git push origin feat-amazing-feature`)
6. Open a Pull Request linking back to the issue

Found a security issue? Please report it privately — see the
[security policy](.github/SECURITY.md).

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) CLI framework
- Uses Tibber's official GraphQL API
- Inspired by Unix philosophy: do one thing well

---

Made with ⚡ by [Kristoffer Risa](https://github.com/kristofferrisa)
