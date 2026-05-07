package cli

import "github.com/spf13/cobra"

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "memex",
		Short:         "Long-running personal memory store for agents",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(NewVersionCmd("0.0.0-dev"))
	root.AddCommand(NewMCPCmd())
	root.AddCommand(NewStatsCmd())
	root.AddCommand(NewStatuslineCmd())
	root.AddCommand(NewDoctorCmd())
	root.AddCommand(NewUpgradeCmd())
	root.AddCommand(NewPurgeCmd())
	root.AddCommand(NewSetupCmd())
	return root
}
