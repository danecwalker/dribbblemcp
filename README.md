# dribbblemcp

MCP server that finds **UI design inspiration on Dribbble** and returns shot images the model can actually see.

Uses your local **Chrome/Chromium** via [chromedp](https://github.com/chromedp/chromedp) — no paid search API keys, no separate browser download. Official Dribbble API v2 only exposes the authenticated user’s own shots, so public inspiration search is done via the public website.

## Tools

| Tool | Purpose |
|------|---------|
| `search_shots` | Free-text search (e.g. `"fintech dashboard dark mode"`) → metadata + preview images |
| `get_shot` | Load one shot by URL or ID → high-res image + description / designer / tags |
| `search_by_tag` | Browse a Dribbble tag (`dashboard`, `saas`, `mobile-app`, …) |

## Requirements

- Go 1.22+
- Google Chrome or Chromium installed (macOS Chrome at the default path works out of the box)

## Install

```bash
# From this repo
make build
make install              # copies binary to ~/.local/bin
```

### Grok / Claude / Cursor config

```toml
# ~/.grok/config.toml  (or .grok/config.toml in a project)
[mcp_servers.dribbble]
command = "/Users/YOU/.local/bin/dribbblemcp"
# or: command = "/Users/YOU/projects/dribbblemcp/bin/dribbblemcp"
enabled = true
startup_timeout_sec = 60   # cold Chromium launch can be slow
tool_timeout_sec = 120
```

CLI equivalent:

```bash
grok mcp add dribbble -- /Users/YOU/.local/bin/dribbblemcp
```

Optional env:

| Variable | Effect |
|----------|--------|
| `CHROME_PATH` | Absolute path to Chrome/Chromium binary |
| `DRIBBBLE_MCP_HEADED=1` | Run Chrome headed (debug) |

## Skill

A companion skill lives at:

```
.grok/skills/dribbble-inspiration/SKILL.md
```

It teaches the agent **when** to pull Dribbble refs, how to write good queries, and how to turn visual observations into original designs (not copies).

## Example flow

1. `search_shots` query=`"SaaS settings page dark mode"`, limit=`6`
2. Inspect returned images for layout / density / hierarchy
3. `get_shot` on the best 1–2 URLs for high-res study
4. Extract principles → apply to your design (colors, spacing scale, component structure)

## Notes & limits

- **Personal design-inspiration use.** Dribbble’s terms restrict scraping and require API-only access for products that redistribute their data. This server is intended as a local assistant tool, not a public mirror of Dribbble.
- Rate-limit yourself. Each call launches page navigations; keep `limit` low (4–8).
- AWS WAF occasionally challenges automated browsers. If a call returns zero results, retry once; headed mode (`DRIBBBLE_MCP_HEADED=1`) can help diagnose.
- Always credit designers and link the original shot when presenting inspiration.

## Development

```bash
make build
make test
go run ./cmd/dribbblemcp   # speaks MCP over stdio
```

## License

MIT
