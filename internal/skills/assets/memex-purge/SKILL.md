---
name: memex-purge
description: |
  Purge the memex knowledge base and session analytics.
  Permanently deletes all indexed content and stats. Irreversible.
  Trigger: /memex:memex-purge
user-invocable: true
---

# memex purge

Permanently delete the memex KB SQLite file and the session analytics SQLite file.

## Instructions

1. **Warn the user** — this is irreversible. Both files are deleted:
   - The FTS5 knowledge base (everything indexed via `memex_index`, `memex_fetch_and_index`, `memex_batch_execute`)
   - The session analytics database (per-tool savings stats)
2. Call the `memex_purge` MCP tool with `{"confirm": true}`.
   - Calling it without `confirm` (or with `confirm: false`) returns the list of paths that *would* be deleted, without touching them — useful for a dry run.
3. Display the result verbatim — the response lists exactly which paths were deleted and which were already absent.
4. **Fallback** (only if the MCP tool is unavailable): run `memex purge --yes` via Bash. The Go binary deletes the same files — no Node.js or TypeScript runtime is involved.

## When to use

- The KB contains stale or incorrect content polluting search results.
- Switching between unrelated projects in the same session.
- Wanting a completely fresh start for this project.

## Important

- `memex_purge` (or `memex purge`) is the **only** way to delete this data. There is no other mechanism.
- `memex_stats` is read-only — it shows statistics, never resets them.
- `/clear` and `/compact` do NOT affect any memex data.
- There is no undo. Re-index content if you need it again.
