---
name: memex-doctor
description: |
  Run memex diagnostics. Checks the Go runtime, SQLite FTS5 support,
  KB and analytics files, and host hooks via the memex_doctor MCP tool.
  Trigger: /memex:memex-doctor
user-invocable: true
---

# memex doctor

Run diagnostics and display the results directly in the conversation.

## Instructions

1. Call the `memex_doctor` MCP tool (no arguments). It runs every check in-process and returns a plain-text status report.
2. Display the result verbatim. Lines are pre-formatted with `[OK]` / `[WARN]` / `[FAIL]` prefixes — renderer-safe (no markdown task-list syntax) for cross-client compatibility.
3. **Fallback** (only if the MCP tool is unavailable — e.g. memex not registered as an MCP server in this host): run `memex doctor` via Bash. The Go binary is the single source of truth — there is no Node.js or TypeScript runtime involved.
