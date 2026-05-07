package hooks

// Routing-block strings injected by the PreToolUse router. These mirror the
// canonical guidance in `internal/routing/claude-code.md` but are trimmed for
// per-event advisory contexts (no markdown chrome that would render badly
// in additionalContext or modifiedToolInput echo strings).
//
// Naming invariant: every string SHALL reference at least one `memex_*` tool
// so the agent learns the correct alternative. The unit test in
// routing_block_test.go enforces this.

// SubagentRoutingBlock is the full guidance string appended to subagent prompts.
// Subagents inherit the rules so they don't dump raw output back into the
// parent context. Distinct from RoutingBlock() in guidance.go which is the
// SessionStart system-prompt block.
const SubagentRoutingBlock = `

memex routing rules apply to this subagent. To keep parent-context lean:
- Web: use memex_fetch_and_index(url, source) then memex_search(queries: [...]).
- Bash > 20 lines stdout: use memex_execute(language, code) — only stdout enters context.
- File analysis: use memex_execute_file(path, language, code) instead of Read.
- Grep across many files: use memex_execute(language: "shell", code: "grep ...").
- Do NOT WebFetch, curl, or wget. They are intercepted; retries fail the same way.
`

// BashGuidance is shown the first time a non-blocked Bash command runs in a
// session. It nudges the agent toward sandbox alternatives without blocking.
const BashGuidance = `memex: Bash output stays in your context. ` +
	`For >20 lines, prefer memex_execute(language, code) — only stdout enters context. ` +
	`Or memex_batch_execute(commands, queries) to run several commands and search in one call. ` +
	`Bash is fine for git, mkdir, rm, mv, ls, cd, and short outputs.`

// ReadGuidance is shown the first time Read fires in a session. Read is fine
// for editing; for analysis the sandbox is cheaper.
const ReadGuidance = `memex: Read pulls full file contents into your context. ` +
	`If you only need to analyse / count / summarise, use memex_execute_file(path, language, code) ` +
	`and console.log() only the answer. Read is correct when the next step is Edit.`

// GrepGuidance is shown the first time Grep fires in a session. Large grep
// hits dump straight into context.
const GrepGuidance = `memex: Grep results enter your context line-for-line. ` +
	`For large result sets prefer memex_execute(language: "shell", code: "grep ... | head -N") ` +
	`or memex_execute(language: "javascript", code: "...") to filter in the sandbox.`

// WebFetchDenyReason is the deny-reason text emitted when a WebFetch call is
// intercepted. Embeds the URL when the input contained one so the agent can
// re-issue the call against memex_fetch_and_index.
func WebFetchDenyReason(url string) string {
	if url == "" {
		return `memex: WebFetch is blocked. Use memex_fetch_and_index(url, source) to fetch and index, ` +
			`then memex_search(queries: [...]) to query. Or memex_execute(language, code) to fetch ` +
			`and console.log() only what you need. Do NOT retry with WebFetch, curl, or wget.`
	}
	return `memex: WebFetch is blocked. Use memex_fetch_and_index(url: "` + url + `", source: "...") ` +
		`then memex_search(queries: [...]) to query. Or memex_execute(language, code) to fetch and ` +
		`console.log() only what you need. Do NOT retry with WebFetch, curl, or wget.`
}

// CurlBlockEcho is the modifiedToolInput.command emitted when a dangerous
// curl/wget invocation is intercepted. The shell echo lands in the agent's
// context as the "result" of running curl, so the message must be a complete
// instruction.
const CurlBlockEcho = `echo "memex: curl/wget is blocked. Use memex_execute(language, code) to fetch ` +
	`and console.log() only the answer, or memex_fetch_and_index(url, source) for HTML/markdown to query later. ` +
	`Pure JS, no npm deps. Do NOT retry with curl or wget."`

// InlineHTTPBlockEcho is the modifiedToolInput.command emitted when an inline
// HTTP call (fetch(, requests.get(, http.get() is detected inside a Bash
// command's code/heredoc.
const InlineHTTPBlockEcho = `echo "memex: Inline HTTP is blocked. Use memex_execute(language, code) ` +
	`to fetch in the sandbox and console.log() only the result. Pure JS, no npm deps. Do NOT retry with Bash."`

// BuildToolBlockEcho is emitted when gradle/mvn/sbt is detected. Build tools
// produce extremely verbose output that should never enter the parent context.
const BuildToolBlockEcho = `echo "memex: Build tool is redirected. Use memex_execute(language: \"shell\", ` +
	`code: \"<your build command> 2>&1 | tail -30\") so only the tail / errors enter context. ` +
	`Do NOT retry with Bash."`
