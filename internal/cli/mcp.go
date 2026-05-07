package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dreamware-nz/memex/internal/adapters"
	"github.com/dreamware-nz/memex/internal/adapters/claudecode"
	"github.com/dreamware-nz/memex/internal/adapters/codex"
	"github.com/dreamware-nz/memex/internal/adapters/cursor"
	"github.com/dreamware-nz/memex/internal/adapters/geminicli"
	"github.com/dreamware-nz/memex/internal/adapters/vscodecopilot"
	"github.com/dreamware-nz/memex/internal/analytics"
	"github.com/dreamware-nz/memex/internal/fetch"
	"github.com/dreamware-nz/memex/internal/kb"
	"github.com/dreamware-nz/memex/internal/mcp"
	"github.com/dreamware-nz/memex/internal/mcp/tools"
	"github.com/dreamware-nz/memex/internal/sandbox"
	"github.com/spf13/cobra"
)

const searchLimit = 5

// sandboxExecutor adapts internal/sandbox to the tools.Executor interface,
// dispatching on the requested language.
type sandboxExecutor struct{}

// mcpRunOpts lifts sandbox-level output caps so the MCP routeOutput
// dispatcher (5 KB intent / 100 KB pointer) controls truncation rather than
// the inner sandbox safety net. MaxBytes still bounds per-stream reads.
func mcpRunOpts() sandbox.RunOptions {
	return sandbox.RunOptions{
		MaxBytes:       4 * 1024 * 1024,
		MaxOutputBytes: 4 * 1024 * 1024,
		MaxLines:       1_000_000,
	}
}

func (sandboxExecutor) Execute(language, code string) (string, string, int, error) {
	ctx := context.Background()
	var (
		res sandbox.Result
		err error
	)
	switch language {
	case "python", "python3":
		res, err = sandbox.RunPython(ctx, code, mcpRunOpts())
	case "node", "javascript", "js":
		res, err = sandbox.RunNode(ctx, code, mcpRunOpts())
	case "shell", "sh", "bash":
		opts := mcpRunOpts()
		opts.Command = []string{"sh", "-c", code}
		res, err = sandbox.Run(ctx, opts)
	default:
		return "", "", 0, fmt.Errorf("unsupported language: %s", language)
	}
	if err != nil {
		return "", "", 0, err
	}
	return res.Stdout, res.Stderr, res.ExitCode, nil
}

// sandboxFileExecutor adapts sandbox.RunFile to tools.FileExecutor. The
// language argument is informational; sandbox.RunFile re-detects from the
// path extension to keep behaviour consistent with bare-CLI execution.
type sandboxFileExecutor struct{}

func (sandboxFileExecutor) ExecuteFile(path, language string) (string, string, int, error) {
	_ = language
	res, err := sandbox.RunFile(context.Background(), path, mcpRunOpts())
	if err != nil {
		return "", "", 0, err
	}
	return res.Stdout, res.Stderr, res.ExitCode, nil
}

// kbSearcher adapts a *kb.Store to the tools.Searcher interface by hardcoding
// the per-call result limit and discarding errors (best-effort search; an
// empty result set is the right "no hits" signal for the tool).
type kbSearcher struct {
	store *kb.Store
}

func (s *kbSearcher) Search(queries []string) []kb.SearchResult {
	out, err := s.store.Search(queries, searchLimit)
	if err != nil {
		return nil
	}
	return out
}

func defaultKBPath() (string, error) {
	if v := os.Getenv("CTX_DB"); v != "" {
		return v, nil
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "memex", "kb.sqlite"), nil
}

// httpFetcher adapts fetch.Get to the tools.Fetcher interface used by FetchTool.
type httpFetcher struct{}

func (httpFetcher) Get(url string) ([]byte, string, error) {
	resp, err := fetch.Get(context.Background(), url, fetch.Options{})
	if err != nil {
		return nil, "", err
	}
	return resp.Body, resp.ContentType, nil
}

// htmlConverter adapts fetch.HTMLToMarkdown to the tools.Converter interface.
// baseURL is empty because FetchTool's Converter contract doesn't pass it
// through; relative-link rewriting falls back to bare paths.
type htmlConverter struct{}

func (htmlConverter) Convert(html string) (string, error) {
	return fetch.HTMLToMarkdown([]byte(html), "")
}

// adapterFactories maps a PlatformID to a constructor — the same pattern
// setup.go uses, factored here so the MCP doctor tool can pick a host
// adapter without importing the setup command.
var adapterFactories = map[string]func() adapters.Adapter{
	"claude-code":    func() adapters.Adapter { return claudecode.New() },
	"codex":          func() adapters.Adapter { return codex.New() },
	"cursor":         func() adapters.Adapter { return cursor.New() },
	"gemini-cli":     func() adapters.Adapter { return geminicli.New() },
	"vscode-copilot": func() adapters.Adapter { return vscodecopilot.New() },
}

// hostSettingsPath returns the host agent's settings.json path used by the
// memex_doctor "host settings" check. Detection mirrors `memex setup`:
// adapters.DetectPlatform picks a platform, then SettingsPath on that
// adapter resolves the file. Returns "" if the detected platform has no
// known adapter — doctor handles empty as a WARN.
func hostSettingsPath() string {
	det := adapters.DetectPlatform()
	make, ok := adapterFactories[det.Platform]
	if !ok {
		return ""
	}
	return make().SettingsPath()
}

// memexBinaryVersion is the version string the memex_doctor tool reports.
// The CLI overrides it via SetMCPVersion before adding NewMCPCmd to root,
// so the same version that `memex version` prints is the one MCP reports.
var memexBinaryVersion = "0.0.0-dev"

// SetMCPVersion overrides the version string memex_doctor includes in its
// report. Called from cmd/memex/main.go alongside NewVersionCmd so MCP and
// CLI agree on identity.
func SetMCPVersion(v string) { memexBinaryVersion = v }

func NewMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run the MCP stdio JSON-RPC server",
		RunE: func(cmd *cobra.Command, args []string) error {
			kbPath, err := defaultKBPath()
			if err != nil {
				return err
			}
			store, err := kb.Open(kbPath)
			if err != nil {
				return fmt.Errorf("open kb: %w", err)
			}
			defer store.Close()

			analyticsPath, err := defaultAnalyticsPath()
			if err != nil {
				return err
			}

			srv := mcp.New()
			searcher := &kbSearcher{store: store}
			srv.Register(&tools.SearchTool{Searcher: searcher})
			srv.Register(&tools.IndexTool{Indexer: store})
			srv.Register(&tools.ExecuteTool{Executor: sandboxExecutor{}, Store: store})
			srv.Register(&tools.ExecuteFileTool{Executor: sandboxFileExecutor{}, Store: store})
			srv.Register(&tools.BatchTool{Executor: sandboxExecutor{}, Indexer: store, Searcher: searcher})
			srv.Register(&tools.FetchTool{Fetcher: httpFetcher{}, Converter: htmlConverter{}, Indexer: store})

			srv.Register(&tools.DoctorTool{
				Version:       memexBinaryVersion,
				KBPath:        kbPath,
				AnalyticsPath: analyticsPath,
				SettingsPath:  hostSettingsPath(),
			})
			srv.Register(&tools.UpgradeTool{Command: upgradeCmd})
			srv.Register(&tools.PurgeTool{Paths: []string{kbPath, analyticsPath}})
			srv.Register(&tools.StatsTool{
				AnalyticsPath: analyticsPath,
				Open: func(p string) (tools.StatsReader, error) {
					return analytics.Open(p)
				},
			})
			return srv.Serve(os.Stdin, os.Stdout)
		},
	}
}
