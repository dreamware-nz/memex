# memex

Go port of [context-mode](../context-mode). The reference TypeScript implementation lives in `../context-mode/src/`. Specs and change proposals live in `../openspec/`.

## Layout

- `cmd/memex/` — CLI entrypoint
- `internal/kb/` — SQLite + FTS5 knowledge base (index + search)
- `internal/sandbox/` — sandboxed code executor (shell, python, node)
- `internal/mcp/` — stdio JSON-RPC MCP server + tool dispatcher
- `internal/hooks/` — agent hook subcommands (PreToolUse, PostToolUse, SessionStart, PreCompact)
- `internal/adapters/` — per-agent install/config (claude-code, gemini-cli, …)
- `internal/session/` — session event tracking (file edits, git ops, tasks, errors)
- `internal/fetch/` — HTTP fetch + HTML→markdown conversion
- `internal/skills/` — embedded skill markdown bundle (`assets/`) + `Write`/`Remove` helpers

## Build

```
cd ctx && go build ./cmd/memex
```

## Quickstart: install host integration

`memex setup install` installs hooks, registers the plugin manifest, and
unpacks the embedded skill bundle into the platform's plugin directory.
The platform is auto-detected; pass `--platform <id>` to override.

```
./memex setup install                       # auto-detect host
./memex setup install --platform claude-code
./memex setup validate --platform claude-code
./memex setup uninstall --platform claude-code
```

Supported platform IDs: `claude-code`, `codex`, `cursor`, `gemini-cli`,
`vscode-copilot`. The skill snapshot lives at
`internal/skills/assets/`; refresh procedure is in
`internal/skills/assets/UPSTREAM-CREDITS.md`.

## Roadmap

See [`../openspec/ROADMAP.md`](../openspec/ROADMAP.md) for the dependency-ordered list of bite-size openspec changes.
