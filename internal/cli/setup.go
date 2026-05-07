package cli

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/dreamware-nz/memex/internal/adapters"
	"github.com/dreamware-nz/memex/internal/adapters/claudecode"
	"github.com/dreamware-nz/memex/internal/adapters/codex"
	"github.com/dreamware-nz/memex/internal/adapters/cursor"
	"github.com/dreamware-nz/memex/internal/adapters/geminicli"
	"github.com/dreamware-nz/memex/internal/adapters/vscodecopilot"
)

// supportedPlatforms maps the platform IDs printed by DetectPlatform to
// adapter constructors. The list is the source of truth for the
// `--platform` flag's enumeration.
var supportedPlatforms = map[string]func() adapters.Adapter{
	"claude-code":    func() adapters.Adapter { return claudecode.New() },
	"codex":          func() adapters.Adapter { return codex.New() },
	"cursor":         func() adapters.Adapter { return cursor.New() },
	"gemini-cli":     func() adapters.Adapter { return geminicli.New() },
	"vscode-copilot": func() adapters.Adapter { return vscodecopilot.New() },
}

// supportedPlatformIDs returns the adapter platform IDs in stable lexical
// order, used in user-facing error messages.
func supportedPlatformIDs() []string {
	ids := make([]string, 0, len(supportedPlatforms))
	for k := range supportedPlatforms {
		ids = append(ids, k)
	}
	sort.Strings(ids)
	return ids
}

// resolveAdapter returns the adapter for explicitID, or — when
// explicitID is empty — the adapter for the auto-detected platform.
// Unknown IDs (or auto-detected IDs not in supportedPlatforms) yield a
// user-facing error listing the supported set.
func resolveAdapter(explicitID string) (adapters.Adapter, string, error) {
	id := explicitID
	if id == "" {
		id = adapters.DetectPlatform().Platform
	}
	ctor, ok := supportedPlatforms[id]
	if !ok {
		return nil, id, fmt.Errorf(
			"unsupported platform %q (supported: %v)",
			id, supportedPlatformIDs(),
		)
	}
	return ctor(), id, nil
}

// NewSetupCmd returns the top-level `memex setup` command tree.
func NewSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Install, uninstall, or validate memex host integration",
	}
	var platform string
	cmd.PersistentFlags().StringVar(&platform, "platform", "",
		"override platform detection (one of: "+
			fmt.Sprintf("%v", supportedPlatformIDs())+")")

	cmd.AddCommand(newSetupInstallCmd(&platform))
	cmd.AddCommand(newSetupUninstallCmd(&platform))
	cmd.AddCommand(newSetupValidateCmd(&platform))
	return cmd
}

func newSetupInstallCmd(platform *string) *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install hooks, plugin manifest, and skills for the detected host",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, id, err := resolveAdapter(*platform)
			if err != nil {
				return err
			}
			binary, err := os.Executable()
			if err != nil {
				return fmt.Errorf("locate own binary: %w", err)
			}
			if err := a.Install(binary); err != nil {
				return fmt.Errorf("install failed: %w", err)
			}
			emitInstallActions(cmd.OutOrStdout(), id)
			return nil
		},
	}
}

func newSetupUninstallCmd(platform *string) *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove memex hooks, plugin manifest entry, and skill bundle",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, id, err := resolveAdapter(*platform)
			if err != nil {
				return err
			}
			binary, err := os.Executable()
			if err != nil {
				return fmt.Errorf("locate own binary: %w", err)
			}
			if err := a.Uninstall(binary); err != nil {
				return fmt.Errorf("uninstall failed: %w", err)
			}
			emitUninstallActions(cmd.OutOrStdout(), id)
			return nil
		},
	}
}

func newSetupValidateCmd(platform *string) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Run host integration diagnostics (hooks, plugin, skills)",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, _, err := resolveAdapter(*platform)
			if err != nil {
				return err
			}
			binary, err := os.Executable()
			if err != nil {
				return fmt.Errorf("locate own binary: %w", err)
			}
			results := a.Validate(binary)
			anyFail := false
			for _, r := range results {
				line := fmt.Sprintf("%s %s: %s", r.Status, r.Check, r.Message)
				if r.Status == "fail" && r.Fix != "" {
					line += " (fix: " + r.Fix + ")"
				}
				fmt.Fprintln(cmd.OutOrStdout(), line)
				if r.Status == "fail" {
					anyFail = true
				}
			}
			if anyFail {
				return fmt.Errorf("validate: one or more checks failed")
			}
			return nil
		},
	}
}

func emitInstallActions(w io.Writer, id string) {
	fmt.Fprintf(w, "install hooks (%s)\n", id)
	fmt.Fprintf(w, "install plugin-manifest (%s)\n", id)
	fmt.Fprintf(w, "install skills (%s)\n", id)
}

func emitUninstallActions(w io.Writer, id string) {
	fmt.Fprintf(w, "uninstall hooks (%s)\n", id)
	fmt.Fprintf(w, "uninstall plugin-manifest (%s)\n", id)
	fmt.Fprintf(w, "uninstall skills (%s)\n", id)
}
