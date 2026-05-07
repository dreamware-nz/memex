package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const upgradeCmd = "go install github.com/dreamware-nz/memex/cmd/memex@latest"

func NewUpgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Print the shell command to install the latest memex",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), upgradeCmd)
			return err
		},
	}
}
