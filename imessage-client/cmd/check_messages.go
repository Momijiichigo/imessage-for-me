package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"imessage-client/config"
	"imessage-client/messaging"
	"imessage-client/notifier"
)

func newCheckMessagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check-messages",
		Short: "Poll for unread iMessage messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := config.LoadRegistration(configPath)
			if err != nil {
				return err
			}
			if reg.IsExpired() {
				return fmt.Errorf("registration data expired; regenerate with mac-registration-provider")
			}

			var store messaging.Store
			if storePath != "" {
				store, err = messaging.NewFileStore(storePath)
				if err != nil {
					return fmt.Errorf("failed to initialize store: %w", err)
				}
			} else {
				store = messaging.NewMemoryStore()
			}

			client := messaging.NewClientWithStore(reg, store)
			summaries, err := client.PollUnread(cmd.Context())
			if errors.Is(err, messaging.ErrHandshakeNotImplemented) {
				fmt.Fprintln(cmd.OutOrStdout(), "Handshake not implemented yet.")
				return nil
			} else if errors.Is(err, messaging.ErrInvalidRegistrationData) {
				return fmt.Errorf("registration data missing required fields")
			} else if errors.Is(err, messaging.ErrNotImplemented) {
				fmt.Fprintln(cmd.OutOrStdout(), "Polling not implemented yet.")
				return nil
			} else if err != nil {
				return err
			}

			notifier.PrintSummaries(cmd.OutOrStdout(), summaries)
			return nil
		},
	}

	return cmd
}
