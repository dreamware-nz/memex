# memex

A local context-management daemon for AI coding agents. `memex` ships
three things in one binary:

- An **MCP server** (stdio JSON-RPC) exposing tools that index, search,
  fetch, and execute code through a per-user SQLite/FTS5 knowledge base
  and a sandboxed code executor.
- **PreToolUse / PostToolUse / SessionStart / PreCompact hooks** that
  route an agent's tool calls (WebFetch, Read, Grep, Bash, subagent
  spawns) through the local KB instead of burning context on raw I/O.
- A **`setup install` installer** that wires both into Claude Code,
  Codex, Cursor, Gemini CLI, or VS Code Copilot on the host machine.

## Install on a new machine

Requirements: Go ≥ 1.25 on PATH (Go ≥ 1.21 will auto-fetch the right
toolchain), an HTTP-reachable `proxy.golang.org`, and one of the
supported host agents below.

### One-liner

```sh
curl -fsSL https://raw.githubusercontent.com/dreamware-nz/memex/main/install.sh | sh
```

This runs [`install.sh`](./install.sh) which: builds memex via
`go install`, places it in `$(go env GOBIN || go env GOPATH/bin)`,
wires it into the auto-detected host agent via `memex setup install`,
and runs `memex setup validate` to confirm.

### Manual

```sh
go install github.com/dreamware-nz/memex/cmd/memex@latest
export PATH="$(go env GOBIN || echo $(go env GOPATH)/bin):$PATH"
memex setup install      # auto-detect host; --platform <id> to override
memex setup validate
```

`setup install` records the running binary's absolute path in the
host agent's hook commands and `.mcp.json`, so wherever you put `memex`
is where the agent will find it.

**For Claude Code users: fully quit and reopen Claude Code after
`setup install`.** MCP servers declared in plugin `.mcp.json` files are
loaded only at startup; an in-flight `/clear` will not pick them up.
After restart, the `mcp__plugin_memex_memex__*` tools appear in the
tool list and the PreToolUse hook starts routing through them.

### Supported host platforms

| Platform ID      | Notes                                                |
| ---------------- | ---------------------------------------------------- |
| `claude-code`    | Plugin tree under `~/.claude/plugins/`               |
| `codex`          | `~/.codex/` config + hooks                           |
| `cursor`         | Cursor extension settings                            |
| `gemini-cli`     | `~/.config/gemini/` settings                         |
| `vscode-copilot` | VS Code Copilot user settings                        |

Auto-detection picks the most-recently-used host. Override with
`--platform <id>`.

## Update

```sh
go install github.com/dreamware-nz/memex/cmd/memex@latest
memex setup install     # re-stamps hook & .mcp.json paths
```

Then **fully quit and reopen Claude Code** (or your host agent) so the
MCP server reloads the new binary — an in-flight session keeps running
the old one until restart.

`memex_upgrade` (the MCP tool) returns this same command so an agent
can self-upgrade on request.

## Uninstall

```sh
memex setup uninstall            # remove hooks, plugin manifest, skills
rm "$(which memex)"              # remove the binary
```

The local SQLite KB at `~/.local/share/memex/kb.sqlite` and analytics
at `~/.local/share/memex/session/analytics.db` are not deleted by
`uninstall`. To remove them, call the `memex_purge` MCP tool with
`confirm: true`, or delete the files by hand.

## Layout

- `cmd/memex/` — CLI entrypoint
- `internal/kb/` — SQLite + FTS5 knowledge base (index + search)
- `internal/sandbox/` — sandboxed code executor (shell, python, node)
- `internal/mcp/` — stdio JSON-RPC MCP server + tool dispatcher
- `internal/hooks/` — agent hook subcommands and PreToolUse routing
- `internal/adapters/` — per-host install / config / detection
- `internal/session/` — session event tracking
- `internal/fetch/` — HTTP fetch + HTML→Markdown conversion
- `internal/skills/` — embedded skill bundle (`assets/`) + writers

## Build from source

```sh
git clone https://github.com/dreamware-nz/memex.git
cd memex
go build ./cmd/memex      # produces ./memex in repo root
go test ./...
```

## License

TBD.
