package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func stubRun(name string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return NewExitError(ExitRuntime, fmt.Sprintf("command %q is not fully wired yet", name))
	}
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Interactive setup (backend, encryption, path map)",
		RunE:  stubRun("init"),
	}
}

func newListCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List local agent sessions",
		RunE:  stubRun("list"),
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().String("agent", "all", "agent filter: claude|codex|all")
	return cmd
}

func newStatusCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Compare local vs remote",
		RunE:  stubRun("status"),
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	return cmd
}

func newDiffCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show pending change metadata",
		RunE:  stubRun("diff"),
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().String("agent", "", "agent filter")
	cmd.Flags().String("session", "", "session id")
	return cmd
}

func newPushCmd() *cobra.Command {
	var asJSON, dryRun bool
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Encrypt and upload local sessions",
		RunE:  stubRun("push"),
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "plan only")
	cmd.Flags().String("agent", "", "agent filter")
	cmd.Flags().String("session", "", "session id")
	cmd.Flags().Bool("all", false, "all sessions")
	return cmd
}

func newPullCmd() *cobra.Command {
	var asJSON, dryRun bool
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Download, decrypt, and restore sessions",
		RunE:  stubRun("pull"),
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "plan only")
	cmd.Flags().String("agent", "", "agent filter")
	cmd.Flags().String("session", "", "session id")
	cmd.Flags().Bool("all", false, "all sessions")
	return cmd
}

func newConflictsCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "conflicts",
		Short: "List and resolve sync conflicts",
	}
	var asJSON bool
	list := &cobra.Command{
		Use:   "list",
		Short: "List conflicts",
		RunE:  stubRun("conflicts list"),
	}
	list.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	show := &cobra.Command{
		Use:   "show <id>",
		Short: "Show conflict metadata",
		Args:  cobra.ExactArgs(1),
		RunE:  stubRun("conflicts show"),
	}
	show.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	resolve := &cobra.Command{
		Use:   "resolve <id>",
		Short: "Resolve a conflict",
		Args:  cobra.ExactArgs(1),
		RunE:  stubRun("conflicts resolve"),
	}
	resolve.Flags().Bool("keep-local", false, "keep local revision")
	resolve.Flags().Bool("keep-remote", false, "keep remote revision")
	resolve.Flags().Bool("keep-both", false, "keep both under forked identity")
	root.AddCommand(list, show, resolve)
	return root
}

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate shell completion scripts",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return NewExitError(ExitUsage, "unsupported shell")
			}
		},
	}
}
