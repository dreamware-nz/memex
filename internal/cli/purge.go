package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func NewPurgeCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "purge",
		Short: "Delete the KB and session analytics SQLite files",
		RunE: func(cmd *cobra.Command, args []string) error {
			kbPath, err := defaultKBPath()
			if err != nil {
				return err
			}
			analyticsPath, err := defaultAnalyticsPath()
			if err != nil {
				return err
			}

			if !yes {
				fmt.Fprint(cmd.ErrOrStderr(), "Are you sure? [y/N]: ")
				line, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
				ans := strings.ToLower(strings.TrimSpace(line))
				if ans != "y" && ans != "yes" {
					fmt.Fprintln(cmd.ErrOrStderr(), "Aborted.")
					return nil
				}
			}

			var deleted []string
			for _, p := range []string{kbPath, analyticsPath} {
				if _, err := os.Stat(p); err == nil {
					if err := os.Remove(p); err != nil {
						return fmt.Errorf("remove %s: %w", p, err)
					}
					deleted = append(deleted, p)
				}
			}

			if len(deleted) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Nothing to purge.")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Purged: %s\n", strings.Join(deleted, ", "))
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation prompt")
	return cmd
}
