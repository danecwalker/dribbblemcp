# dribbblemcp

[![CI](https://github.com/danecwalker/dribbblemcp/actions/workflows/ci.yml/badge.svg)](https://github.com/danecwalker/dribbblemcp/actions/workflows/ci.yml)
[![Release](https://github.com/danecwalker/dribbblemcp/actions/workflows/release.yml/badge.svg)](https://github.com/danecwalker/dribbblemcp/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

MCP server that finds **UI design inspiration on Dribbble** and returns shot images your agent can actually see.

Uses your local **Chrome/Chromium** via [chromedp](https://github.com/chromedp/chromedp) — no paid search API keys. Official Dribbble API v2 only exposes the authenticated user’s own shots, so public inspiration search goes through the website in a real browser.

## Tools

| Tool | Purpose |
|------|---------|
| `search_shots` | Free-text search (e.g. `"fintech dashboard dark mode"`) → metadata + preview images |
| `get_shot` | Load one shot by URL or ID → high-res image + description / designer / tags |
| `search_by_tag` | Browse a Dribbble tag (`dashboard`, `saas`, `mobile-app`, …) |

## Requirements

- **Chrome or Chromium** installed (macOS default Chrome path works out of the box)
- Network access to `dribbble.com` and `cdn.dribbble.com`
- For building from source: **Go 1.26+**

---

## Install

### One-liner (macOS / Linux) — recommended

Downloads the latest GitHub Release binary for your OS/arch into `~/.local/bin`:

```bash
curl -fsSL https://raw.githubusercontent.com/danecwalker/dribbblemcp/main/install.sh | bash
```

Options:

```bash
# Specific version
curl -fsSL https://raw.githubusercontent.com/danecwalker/dribbblemcp/main/install.sh | bash -s -- --version v0.1.0

# Custom install directory
curl -fsSL https://raw.githubusercontent.com/danecwalker/dribbblemcp/main/install.sh | bash -s -- --dir /usr/local/bin
```

Ensure `~/.local/bin` is on your `PATH`:

```bash
# fish
fish_add_path $HOME/.local/bin

# bash / zsh
export PATH="$HOME/.local/bin:$PATH"
```

### Go install

```bash
go install github.com/danecwalker/dribbblemcp/cmd/dribbblemcp@latest
```

Binary lands in `$(go env GOPATH)/bin` (often `~/go/bin`).

### Pre-built binaries

1. Open [Releases](https://github.com/danecwalker/dribbblemcp/releases/latest)
2. Download the archive for your platform:

   | Platform | Archive |
   |----------|---------|
   | macOS Apple Silicon | `dribbblemcp_Darwin_arm64.tar.gz` |
   | macOS Intel | `dribbblemcp_Darwin_x86_64.tar.gz` |
   | Linux x86_64 | `dribbblemcp_Linux_x86_64.tar.gz` |
   | Linux arm64 | `dribbblemcp_Linux_arm64.tar.gz` |
   | Windows x86_64 | `dribbblemcp_Windows_x86_64.zip` |

3. Extract and move `dribbblemcp` onto your `PATH`
4. Verify checksums with `checksums.txt` from the same release

```bash
# Example (macOS arm64)
curl -fsSL -O https://github.com/danecwalker/dribbblemcp/releases/latest/download/dribbblemcp_Darwin_arm64.tar.gz
tar -xzf dribbblemcp_Darwin_arm64.tar.gz
install -m 755 dribbblemcp ~/.local/bin/dribbblemcp
dribbblemcp --version
```

### From source

```bash
git clone https://github.com/danecwalker/dribbblemcp.git
cd dribbblemcp
make build          # → bin/dribbblemcp
make install        # → ~/.local/bin/dribbblemcp
```

---

## Configure your MCP client

### Grok

```bash
grok mcp add dribbble -- $(which dribbblemcp)
```

Or edit `~/.grok/config.toml`:

```toml
[mcp_servers.dribbble]
command = "/Users/YOU/.local/bin/dribbblemcp"
enabled = true
startup_timeout_sec = 60   # cold Chrome launch can be slow
tool_timeout_sec = 120
```

### Claude Desktop

`~/Library/Application Support/Claude/claude_desktop_config.json` (macOS):

```json
{
  "mcpServers": {
    "dribbble": {
      "command": "/Users/YOU/.local/bin/dribbblemcp"
    }
  }
}
```

### Cursor

MCP settings → add server:

```json
{
  "mcpServers": {
    "dribbble": {
      "command": "/Users/YOU/.local/bin/dribbblemcp"
    }
  }
}
```

### Environment variables

| Variable | Effect |
|----------|--------|
| `CHROME_PATH` | Absolute path to Chrome/Chromium binary |
| `DRIBBBLE_MCP_HEADED=1` | Run Chrome headed (debug WAF / layout issues) |

---

## Skill

Companion skill (query craft + how to turn shots into original design direction):

```
.grok/skills/dribbble-inspiration/SKILL.md
```

Copy into a project’s `.grok/skills/` or `~/.grok/skills/dribbble-inspiration/`. Trigger with design-inspiration asks or `/dribbble-inspiration`.

---

## Example flow

1. `search_shots` query=`"SaaS settings page dark mode"`, limit=`6`
2. Inspect returned images for layout / density / hierarchy
3. `get_shot` on the best 1–2 URLs for high-res study
4. Extract principles → apply to **your** design (do not clone)

---

## Development

```bash
make build
make test                 # unit tests
make test-integration     # live Dribbble (Chrome + network)
make doctor               # build + test + --version
```

### Release (maintainers)

Releases are automated with [GoReleaser](https://goreleaser.com) on version tags:

```bash
# Local snapshot (no GitHub publish)
make release-snapshot

# Publish: push a tag — GitHub Actions runs .github/workflows/release.yml
git tag v0.1.0
git push origin v0.1.0
```

Artifacts: multi-arch archives + `checksums.txt` on the [Releases](https://github.com/danecwalker/dribbblemcp/releases) page. The install script always points at the latest release.

Config: [`.goreleaser.yaml`](.goreleaser.yaml) · workflows: [CI](.github/workflows/ci.yml), [Release](.github/workflows/release.yml).

---

## Notes & limits

- **Personal design-inspiration use.** Dribbble’s terms restrict scraping and prefer API-only redistribution. This server is a local assistant tool, not a public mirror of Dribbble.
- Rate-limit yourself. Each call navigates pages; keep `limit` low (4–8).
- AWS WAF occasionally challenges automation. If a call returns zero results, retry once; headed mode (`DRIBBBLE_MCP_HEADED=1`) helps diagnose.
- Always credit designers and link the original shot when presenting inspiration.

## License

[MIT](LICENSE)
