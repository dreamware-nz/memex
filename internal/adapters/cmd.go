package adapters

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewDetectCmd returns the `ctx detect` debug subcommand which prints the
// current platform detection signal in a single grep-friendly line.
func NewDetectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "detect",
		Short: "Print the detected host platform and confidence",
		RunE: func(cmd *cobra.Command, args []string) error {
			sig := DetectPlatform()
			_, err := fmt.Fprintf(cmd.OutOrStdout(),
				"platform=%s confidence=%s reason=%s\n",
				sig.Platform, sig.Confidence, sig.Reason)
			return err
		},
	}
}
