package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dreamware-nz/memex/internal/statusline"
	"github.com/spf13/cobra"
)

// defaultStatsPath resolves the persisted statusline payload path using the
// same precedence as the rest of memex: MEMEX_DATA_DIR → XDG_DATA_HOME →
// ~/.local/share, with the canonical `memex/session/stats.json` suffix.
func defaultStatsPath() (string, error) {
	if v := os.Getenv("MEMEX_DATA_DIR"); v != "" {
		return filepath.Join(v, "memex", "session", "stats.json"), nil
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "memex", "session", "stats.json"), nil
}

func NewStatuslineCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "statusline",
		Short: "Print the Claude Code status bar one-liner",
		// Any read or parse failure exits 0 with no output: a blank status
		// bar beats error noise on every render tick.
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := defaultStatsPath()
			if err != nil {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			payload, err := statusline.ParsePayload(data)
			if err != nil || payload == nil {
				return nil
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), statusline.Render(payload))
			return nil
		},
	}
}
