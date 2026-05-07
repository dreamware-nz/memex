package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Action is the abstract verb a router decision applies to a tool call.
// It mirrors the TS reference's `action` field 1:1.
type Action int

const (
	// ActionPassthrough emits `{}` — the tool call proceeds untouched.
	ActionPassthrough Action = iota
	// ActionDeny blocks the call with a reason naming the memex alternative.
	ActionDeny
	// ActionAsk forces the host to prompt the user before running.
	ActionAsk
	// ActionModify replaces the tool input (typically Bash.command or Agent.prompt).
	ActionModify
	// ActionContext attaches an advisory string the host shows to the model.
	ActionContext
)

// Decision is the structured outcome of routing a single PreToolUse event.
// Only the fields relevant to Action are populated. The formatter in
// `formatClaudeCode` translates this into the Claude Code response shape.
type Decision struct {
	Action        Action
	Reason        string         // Used by ActionDeny / ActionAsk.
	UpdatedInput  map[string]any // Used by ActionModify.
	AdditionalCtx string         // Used by ActionContext.
}

// toolAliases maps platform-specific tool names to canonical Claude Code names.
// Mirrors `TOOL_ALIASES` in `context-mode/hooks/core/routing.mjs`. Unknown
// tools pass through unchanged so future names degrade to passthrough rather
// than being silently mis-routed.
var toolAliases = map[string]string{
	// Gemini CLI / Qwen Code (shared native tool names).
	"run_shell_command": "Bash",
	"read_file":         "Read",
	"read_many_files":   "Read",
	"grep_search":       "Grep",
	"search_file_content": "Grep",
	"web_fetch":         "WebFetch",
	"write_file":        "Write",
	"edit":              "Edit",
	"glob":              "Glob",
	"todo_write":        "TodoWrite",
	"ask_user_question": "AskUserQuestion",
	"list_directory":    "LS",
	"save_memory":       "Memory",
	"skill":             "Skill",
	"exit_plan_mode":    "ExitPlanMode",

	// OpenCode.
	"bash":  "Bash",
	"view":  "Read",
	"grep":  "Grep",
	"fetch": "WebFetch",
	"agent": "Agent",

	// Codex CLI.
	"shell":          "Bash",
	"shell_command":  "Bash",
	"exec_command":   "Bash",
	"container.exec": "Bash",
	"local_shell":    "Bash",
	"grep_files":     "Grep",

	// OpenClaw.
	"exec":   "Bash",
	"read":   "Read",
	"search": "Grep",

	// Cursor.
	"mcp_web_fetch":  "WebFetch",
	"mcp_fetch_tool": "WebFetch",
	"Shell":          "Bash",

	// VS Code Copilot.
	"run_in_terminal": "Bash",

	// Kiro CLI.
	"fs_read":      "Read",
	"fs_write":     "Write",
	"execute_bash": "Bash",
}

// canonicalName returns the Claude Code-canonical tool name for an arbitrary
// platform-specific name, or the input unchanged when no alias is registered.
func canonicalName(name string) string {
	if c, ok := toolAliases[name]; ok {
		return c
	}
	return name
}

// guidanceOnce returns ActionContext with the supplied guidance the FIRST time
// a (sessionID, guidanceType) pair is seen, and ActionPassthrough thereafter.
// Concurrency-safe across hook processes via O_EXCL marker files in tmpdir.
func guidanceOnce(guidanceType, content, sessionID string) Decision {
	var dirName string
	if sessionID != "" {
		dirName = fmt.Sprintf("memex-guidance-s-%s", sanitizeForFilename(sessionID))
	} else {
		dirName = fmt.Sprintf("memex-guidance-pid-%d", os.Getppid())
	}
	dir := filepath.Join(os.TempDir(), dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		if os.Getenv("MEMEX_GUIDANCE_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "guidanceOnce mkdir %s: %v\n", dir, err)
		}
		return Decision{Action: ActionPassthrough}
	}
	marker := filepath.Join(dir, guidanceType+".marker")
	f, err := os.OpenFile(marker, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.Getenv("MEMEX_GUIDANCE_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "guidanceOnce open %s: %v\n", marker, err)
		}
		// Marker exists (already fired) or filesystem rejected the open —
		// either way, fall through silently.
		return Decision{Action: ActionPassthrough}
	}
	_ = f.Close()
	return Decision{Action: ActionContext, AdditionalCtx: content}
}

// sanitizeForFilename strips path separators and other characters that would
// confuse os.MkdirAll if a hostile session ID landed in the payload.
var unsafeFilenameRE = regexp.MustCompile(`[^A-Za-z0-9_.-]`)

func sanitizeForFilename(s string) string {
	return unsafeFilenameRE.ReplaceAllString(s, "_")
}

// stripQuotedContent removes the bodies of single-quoted, double-quoted, and
// backtick-quoted segments so pattern matching can ignore literals like
// `gh issue edit --body "see curl example"`.
func stripQuotedContent(cmd string) string {
	out := cmd
	out = regexp.MustCompile(`'[^']*'`).ReplaceAllString(out, "''")
	out = regexp.MustCompile(`"[^"]*"`).ReplaceAllString(out, `""`)
	out = regexp.MustCompile("`[^`]*`").ReplaceAllString(out, "``")
	return out
}

// stripHeredocs collapses `<<EOF ... EOF` and `<<-EOF ... EOF` blocks so a
// heredoc body containing `fetch(` or `requests.get(` doesn't trigger a false
// positive on the inline-HTTP detector. Go's regexp engine (RE2) does not
// support backreferences, so we scan manually: find each `<<` opener, capture
// the terminator word, then drop everything up to and including the next line
// that contains exactly that word.
var heredocOpenRE = regexp.MustCompile(`<<-?\s*['"]?(\w+)['"]?`)

func stripHeredocs(cmd string) string {
	out := cmd
	for {
		loc := heredocOpenRE.FindStringSubmatchIndex(out)
		if loc == nil {
			return out
		}
		openerStart := loc[0]
		// loc[2:4] = group 1 (the terminator word).
		terminator := out[loc[2]:loc[3]]
		// Find the start of the body: end of the opener line.
		bodyStart := openerStart + (loc[1] - loc[0])
		nl := strings.Index(out[bodyStart:], "\n")
		if nl < 0 {
			// Opener with no body — strip from opener to EOL (which is end).
			return out[:openerStart]
		}
		bodyStart += nl + 1
		// Find the closing line: a line whose trimmed content equals terminator.
		closeStart := -1
		closeEnd := -1
		i := bodyStart
		for i <= len(out) {
			lineEnd := strings.Index(out[i:], "\n")
			var line string
			if lineEnd < 0 {
				line = out[i:]
				lineEnd = len(out) - i
			} else {
				line = out[i : i+lineEnd]
			}
			if strings.TrimSpace(line) == terminator {
				closeStart = i
				closeEnd = i + lineEnd
				if closeEnd < len(out) && out[closeEnd] == '\n' {
					closeEnd++
				}
				break
			}
			if lineEnd < 0 {
				break
			}
			i += lineEnd + 1
		}
		if closeStart < 0 {
			// No terminator found — drop opener line only and continue.
			out = out[:openerStart] + out[bodyStart:]
			continue
		}
		out = out[:openerStart] + out[closeEnd:]
	}
}

// isCurlOrWgetUnsafe returns true if the given command segment invokes
// curl/wget in a way that floods stdout: no file output, redirection to a
// stdout alias, or verbose flags set.
func isCurlOrWgetUnsafe(seg string) bool {
	s := stripQuotedContent(seg)
	if !regexp.MustCompile(`(?:^|[\s;&|(])(curl|wget)(?:\s|$)`).MatchString(s) {
		return false
	}
	// Verbose flags always flood — block regardless of redirection.
	if regexp.MustCompile(`(?:\s|^)(-v|--verbose|--trace|--trace-ascii)\b`).MatchString(s) {
		return true
	}
	// stdout alias output is still a flood.
	if regexp.MustCompile(`(?:\s|^)-[oO]\s+(?:-|/dev/stdout)(?:\s|$)`).MatchString(s) {
		return true
	}
	// File output (`-o file`, `-O file`, `> file`, `>> file`) makes it safe.
	if regexp.MustCompile(`(?:\s|^)-[oO]\s+\S+`).MatchString(s) {
		return false
	}
	if regexp.MustCompile(`>>?\s*\S+`).MatchString(s) {
		return false
	}
	// Bare curl/wget with no redirection — flood.
	return true
}

// splitShellSegments splits on top-level `&&`, `||`, `;` so each can be
// evaluated independently for safety. Quote-aware enough to avoid splitting
// inside strings.
func splitShellSegments(cmd string) []string {
	stripped := stripQuotedContent(cmd)
	idxs := []int{0}
	re := regexp.MustCompile(`&&|\|\||;`)
	for _, m := range re.FindAllStringIndex(stripped, -1) {
		idxs = append(idxs, m[1])
	}
	idxs = append(idxs, len(stripped))
	out := make([]string, 0, len(idxs))
	for i := 0; i+1 < len(idxs); i++ {
		seg := strings.TrimSpace(stripped[idxs[i]:idxs[i+1]])
		if seg != "" {
			out = append(out, seg)
		}
	}
	if len(out) == 0 {
		return []string{strings.TrimSpace(stripped)}
	}
	return out
}

var (
	inlineFetchRE  = regexp.MustCompile(`fetch\s*\(\s*['"]https?://`)
	inlineReqRE    = regexp.MustCompile(`requests\.(get|post|put|patch|delete)\s*\(`)
	inlineHTTPRE   = regexp.MustCompile(`\bhttp\.(get|request)\s*\(`)
	buildToolRE    = regexp.MustCompile(`(?:^|[\s;&|(])(\.\/gradlew|gradlew|gradle|\.\/mvnw|mvnw|mvn|\.\/sbt|sbt)(?:\s|$|;|&|\|)`)
)

// routeBash applies the curl/wget, inline-HTTP, build-tool, and advisory
// branches in that order. Returns ActionPassthrough only if all checks fall
// through and the bash advisory has already fired this session.
func routeBash(input map[string]any, sessionID string) Decision {
	cmd, _ := input["command"].(string)
	if cmd == "" {
		return Decision{Action: ActionPassthrough}
	}

	// curl/wget per-segment.
	for _, seg := range splitShellSegments(cmd) {
		if isCurlOrWgetUnsafe(seg) {
			return Decision{
				Action: ActionModify,
				UpdatedInput: map[string]any{
					"command": CurlBlockEcho,
				},
			}
		}
	}

	// Inline HTTP after stripping heredocs.
	noHere := stripHeredocs(cmd)
	if inlineFetchRE.MatchString(noHere) || inlineReqRE.MatchString(noHere) || inlineHTTPRE.MatchString(noHere) {
		return Decision{
			Action: ActionModify,
			UpdatedInput: map[string]any{
				"command": InlineHTTPBlockEcho,
			},
		}
	}

	// Build tools (with quoted bodies stripped to avoid false positives).
	stripped := stripQuotedContent(cmd)
	if buildToolRE.MatchString(stripped) {
		return Decision{
			Action: ActionModify,
			UpdatedInput: map[string]any{
				"command": BuildToolBlockEcho,
			},
		}
	}

	return guidanceOnce("bash", BashGuidance, sessionID)
}

// promptFields are the keys we'll append the routing block to, in priority
// order. The first one present is rewritten and the rest are left alone.
var promptFields = []string{"prompt", "request", "objective", "question", "query", "task"}

// routeAgent appends the SubagentRoutingBlock to the first prompt-bearing
// field on the Agent payload. If subagent_type is "Bash" it rewrites it to
// "general-purpose" so the subagent doesn't bypass routing by claiming it is
// the bash interpreter itself.
func routeAgent(input map[string]any) Decision {
	updated := make(map[string]any, len(input)+1)
	for k, v := range input {
		updated[k] = v
	}

	for _, field := range promptFields {
		if existing, ok := updated[field].(string); ok {
			updated[field] = existing + SubagentRoutingBlock
			break
		}
	}

	if st, ok := updated["subagent_type"].(string); ok && st == "Bash" {
		updated["subagent_type"] = "general-purpose"
	}

	return Decision{Action: ActionModify, UpdatedInput: updated}
}

// Route is the top-level dispatcher. Given a tool name and input from a
// PreToolUse event, it returns the structured Decision the formatter will
// translate into the Claude Code response shape. Errors NEVER propagate out
// of Route — internal failures degrade to ActionPassthrough so a transient
// I/O issue can't block a legitimate tool call.
func Route(toolName string, toolInput map[string]any, sessionID string) (decision Decision) {
	defer func() {
		if r := recover(); r != nil {
			decision = Decision{Action: ActionPassthrough}
		}
	}()

	if toolInput == nil {
		toolInput = map[string]any{}
	}
	canon := canonicalName(toolName)

	switch canon {
	case "Bash":
		return routeBash(toolInput, sessionID)
	case "Read":
		return guidanceOnce("read", ReadGuidance, sessionID)
	case "Grep":
		return guidanceOnce("grep", GrepGuidance, sessionID)
	case "WebFetch":
		url, _ := toolInput["url"].(string)
		return Decision{Action: ActionDeny, Reason: WebFetchDenyReason(url)}
	case "Agent", "Task":
		return routeAgent(toolInput)
	default:
		return Decision{Action: ActionPassthrough}
	}
}

// formatClaudeCode translates the abstract Decision into the Claude Code
// PreToolUse response shape (`hookSpecificOutput.*`). Passthrough returns the
// zero value so PreToolUseResponse marshals to `{}`.
func formatClaudeCode(d Decision) PreToolUseResponse {
	switch d.Action {
	case ActionDeny:
		return PreToolUseResponse{
			HookSpecificOutput: &PreToolUseHookOutput{
				HookEventName:            "PreToolUse",
				PermissionDecision:       "deny",
				PermissionDecisionReason: d.Reason,
			},
		}
	case ActionAsk:
		return PreToolUseResponse{
			HookSpecificOutput: &PreToolUseHookOutput{
				HookEventName:            "PreToolUse",
				PermissionDecision:       "ask",
				PermissionDecisionReason: d.Reason,
			},
		}
	case ActionModify:
		return PreToolUseResponse{
			HookSpecificOutput: &PreToolUseHookOutput{
				HookEventName:     "PreToolUse",
				ModifiedToolInput: d.UpdatedInput,
			},
		}
	case ActionContext:
		return PreToolUseResponse{
			HookSpecificOutput: &PreToolUseHookOutput{
				HookEventName:     "PreToolUse",
				AdditionalContext: d.AdditionalCtx,
			},
		}
	default:
		return PreToolUseResponse{}
	}
}
