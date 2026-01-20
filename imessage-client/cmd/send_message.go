package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"imessage-client/config"
	"imessage-client/messaging"
)

func newSendMessageCmd() *cobra.Command {
	var chat string
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a message to a chat/recipient",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text := args[0]

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
			if chat == "" {
				return fmt.Errorf("recipient/chat is required (use --chat)")
			}
			if err := client.Send(cmd.Context(), chat, text); err != nil {
				if errors.Is(err, messaging.ErrHandshakeNotImplemented) {
					fmt.Fprintln(cmd.OutOrStdout(), "Handshake not implemented yet.")
					return nil
				} else if errors.Is(err, messaging.ErrNotImplemented) {
					fmt.Fprintln(cmd.OutOrStdout(), "Send not implemented yet.")
					return nil
				}
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Sent (stub).")
			return nil
		},
	}

	cmd.Flags().StringVar(&chat, "chat", "", "Chat/recipient identifier")
	return cmd
}
