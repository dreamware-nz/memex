package hooks

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func NewHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hook",
		Short: "Run a hook handler",
	}
	cmd.AddCommand(NewPreToolUseCmd())
	cmd.AddCommand(NewPostToolUseCmd())
	cmd.AddCommand(NewSessionStartCmd())
	cmd.AddCommand(NewPreCompactCmd())
	return cmd
}

func NewPreCompactCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "precompact",
		Short: "Handle a PreCompact hook event from stdin",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := HandlePreCompact(os.Stdin, os.Stdout); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return nil
		},
	}
}

func NewPreToolUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pretooluse",
		Short: "Handle a PreToolUse hook event from stdin",
		// Exit 1 directly on hook failure — agents reserve non-zero exits for
		// hook crashes, distinct from valid deny decisions which return 0 with
		// `{decision:"deny"}` on stdout.
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := HandlePreToolUse(os.Stdin, os.Stdout); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return nil
		},
	}
}

func NewPostToolUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "posttooluse",
		Short: "Handle a PostToolUse hook event from stdin",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := HandlePostToolUse(os.Stdin, os.Stdout); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return nil
		},
	}
}

func NewSessionStartCmd() *cobra.Command {
	var continueSession bool
	cmd := &cobra.Command{
		Use:   "sessionstart",
		Short: "Handle a SessionStart hook event from stdin",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := HandleSessionStart(os.Stdin, os.Stdout, continueSession); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return nil
		},
	}
	cmd.PersistentFlags().BoolVar(&continueSession, "continue", false,
		"prepend a prior-session recap to the injected system prompt")
	return cmd
}
