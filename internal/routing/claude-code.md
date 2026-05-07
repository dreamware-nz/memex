# memex — MANDATORY routing rules

memex MCP tools available. Rules protect context window from flooding. One unrouted command dumps 56 KB into context.

## Think in Code — MANDATORY

Analyze/count/filter/compare/search/parse/transform data: **write code** via `memex_execute(language, code)`, `console.log()` only the answer. Do NOT read raw data into context. PROGRAM the analysis, not COMPUTE it. Pure JavaScript — Node.js built-ins only (`fs`, `path`, `child_process`). `try/catch`, handle `null`/`undefined`. One script replaces ten tool calls.

## BLOCKED — do NOT attempt

### curl / wget — BLOCKED
Intercepted and replaced with error. Do NOT retry.
Use: `memex_fetch_and_index(url, source)` or `memex_execute(language: "javascript", code: "const r = await fetch(...)")`

### Inline HTTP — BLOCKED
`fetch('http`, `requests.get(`, `requests.post(`, `http.get(`, `http.request(` — intercepted. Do NOT retry.
Use: `memex_execute(language, code)` — only stdout enters context

### WebFetch — BLOCKED
Use: `memex_fetch_and_index(url, source)` then `memex_search(queries)`

## REDIRECTED — use sandbox

### Bash (>20 lines output)
Bash ONLY for: `git`, `mkdir`, `rm`, `mv`, `cd`, `ls`, `npm install`, `pip install`.
Otherwise: `memex_batch_execute(commands, queries)` or `memex_execute(language: "shell", code: "...")`

### Read (for analysis)
Reading to **Edit** → Read correct. Reading to **analyze/explore/summarize** → `memex_execute_file(path, language, code)`.

### Grep (large results)
Use `memex_execute(language: "shell", code: "grep ...")` in sandbox.

## Tool selection

0. **MEMORY**: `memex_search(sort: "timeline")` — after resume, check prior context before asking user.
1. **GATHER**: `memex_batch_execute(commands, queries)` — runs all commands, auto-indexes, returns search. ONE call replaces 30+. Each command: `{label: "header", command: "..."}`.
2. **FOLLOW-UP**: `memex_search(queries: ["q1", "q2", ...])` — all questions as array, ONE call (default relevance mode).
3. **PROCESSING**: `memex_execute(language, code)` | `memex_execute_file(path, language, code)` — sandbox, only stdout enters context.
4. **WEB**: `memex_fetch_and_index(url, source)` then `memex_search(queries)` — raw HTML never enters context.
5. **INDEX**: `memex_index(content, source)` — store in FTS5 for later search.

## Parallel I/O batches

For multi-URL fetches or multi-API calls, **always** include `concurrency: N` (1-8):

- `memex_batch_execute(commands: [3+ network commands], concurrency: 5)` — gh, curl, dig, docker inspect, multi-region cloud queries
- `memex_fetch_and_index(requests: [{url, source}, ...], concurrency: 5)` — multi-URL batch fetch

**Use concurrency 4-8** for I/O-bound work (network calls, API queries). **Keep concurrency 1** for CPU-bound (npm test, build, lint) or commands sharing state (ports, lock files, same-repo writes).

GitHub API rate-limit: cap at 4 for `gh` calls.

## Subagent routing

Routing block auto-injected into subagent prompts. Bash-type subagents upgraded to general-purpose. No manual instruction needed.

## Session Continuity

Skills, roles, and decisions persist for the entire session. Do not abandon them as the conversation grows.

## Memory

Session history is persistent and searchable. On resume, search BEFORE asking the user:

| Need | Command |
|------|---------|
| What were we working on? | `memex_search(queries: ["summary"], source: "compaction", sort: "timeline")` |
| What was the first request? | `memex_search(queries: ["prompt"], source: "user-prompt", sort: "timeline")` |
| What did we decide? | `memex_search(queries: ["decision"], source: "decision", sort: "timeline")` |
| What NOT to repeat? | `memex_search(queries: ["rejected"], source: "rejected-approach")` |
| What constraints exist? | `memex_search(queries: ["constraint"], source: "constraint")` |

DO NOT ask "what were we working on?" — SEARCH FIRST.
If search returns 0 results, proceed as a fresh session.

## memex commands

| Command | Action |
|---------|--------|
| `memex stats` | Call `memex_stats` MCP tool, display full output verbatim |
| `memex doctor` | Call `memex_doctor` MCP tool, run returned shell command, display as checklist |
| `memex upgrade` | Call `memex_upgrade` MCP tool, run returned shell command, display as checklist |
| `memex purge` | Call `memex_purge` MCP tool with confirm: true. Warns before wiping knowledge base. |

After /clear or /compact: knowledge base and session stats preserved. Use `memex purge` to start fresh.
