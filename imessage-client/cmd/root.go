package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var configPath string
var storePath string

func defaultStorePath() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		return ""
	}
	return filepath.Join(base, "imessage-client", "state.json")
}

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "imessage-client",
		Short: "Lightweight iMessage CLI client",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Placeholder interactive mode until real-time session is wired up.
			fmt.Fprintln(cmd.OutOrStdout(), "Interactive mode is not implemented yet.")
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&configPath, "registration", "registration-data.json", "Path to registration data JSON")
	cmd.PersistentFlags().StringVar(&storePath, "store", defaultStorePath(), "Path to state store for unread tracking (\"\" for in-memory)")
	cmd.AddCommand(newCheckMessagesCmd())
	cmd.AddCommand(newSendMessageCmd())

	return cmd
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
