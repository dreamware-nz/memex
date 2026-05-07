package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/dreamware-nz/memex/internal/analytics"
	"github.com/spf13/cobra"
)

// defaultAnalyticsPath resolves the session analytics database location.
//
// Honours MEMEX_DATA_DIR (used by tests and operators that want an isolated
// install) before falling back to XDG_DATA_HOME and finally ~/.local/share —
// the same precedence the rest of memex uses.
func defaultAnalyticsPath() (string, error) {
	if v := os.Getenv("MEMEX_DATA_DIR"); v != "" {
		return filepath.Join(v, "memex", "session", "analytics.db"), nil
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "memex", "session", "analytics.db"), nil
}

func NewStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show per-tool token and USD savings",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := defaultAnalyticsPath()
			if err != nil {
				return err
			}
			if _, err := os.Stat(path); os.IsNotExist(err) {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No stats yet — run a session first.")
				return nil
			} else if err != nil {
				return fmt.Errorf("stat analytics db: %w", err)
			}

			r, err := analytics.Open(path)
			if err != nil {
				return err
			}
			defer r.Close()

			rows, err := r.All()
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "Tool\tCalls\tTokens Saved\tUSD Saved")
			var totalCalls, totalTokens int64
			var totalUSD float64
			for _, row := range rows {
				fmt.Fprintf(tw, "%s\t%d\t%d\t%.4f\n",
					row.Tool, row.Calls, row.TokensSaved, row.USDSaved)
				totalCalls += row.Calls
				totalTokens += row.TokensSaved
				totalUSD += row.USDSaved
			}
			fmt.Fprintln(tw, strings.Repeat("-", 8)+"\t"+
				strings.Repeat("-", 5)+"\t"+
				strings.Repeat("-", 12)+"\t"+
				strings.Repeat("-", 9))
			fmt.Fprintf(tw, "TOTAL\t%d\t%d\t%.4f\n", totalCalls, totalTokens, totalUSD)
			return tw.Flush()
		},
	}
}
