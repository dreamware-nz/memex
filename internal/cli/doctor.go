package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// diagnosticCmd is the shell one-liner emitted by `memex doctor`.  Agents run
// it to gather Go/Node/sqlite-FTS5/memex version info.  See
// openspec/changes/meta-ctx-doctor/design.md for why this is emitted rather
// than executed.
const diagnosticCmd = `go version && node --version && sqlite3 :memory: "CREATE VIRTUAL TABLE _fts5_probe USING fts5(x); SELECT 'fts5: ok';" && memex version`

func NewDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Print a diagnostic shell command for the agent to run",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), diagnosticCmd)
			return err
		},
	}
}
