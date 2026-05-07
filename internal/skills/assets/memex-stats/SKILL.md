---
name: memex-stats
description: |
  Show how much context window memex saved this session.
  Per-tool token and USD savings table. Read-only.
  To wipe data entirely use memex_purge.
  Trigger: /memex:memex-stats
user-invocable: true
---

# memex stats

Show context savings for the current session.

## Instructions

1. Call the `memex_stats` MCP tool (no arguments).
2. **CRITICAL**: paste the entire tool output verbatim into your reply as a fenced code block. Do not summarise, collapse, or paraphrase — the tab-aligned table only reads correctly when copied byte-for-byte.
3. After the table, add one short sentence highlighting the headline number. Examples:
   - "memex saved **12.4×** — most data stayed in the sandbox."
   - "No memex calls this session yet."
4. **Fallback** (only if the MCP tool is unavailable): run `memex stats` via Bash. Same Go binary, same output — no Node.js or TypeScript runtime.

## To wipe data

- `memex_purge(confirm: true)` permanently deletes the KB and analytics files. Use `/memex:memex-purge` for that flow.
