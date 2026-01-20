package notifier

import (
	"fmt"
	"io"
	"time"

	"imessage-client/messaging"
)

func PrintSummaries(w io.Writer, summaries []messaging.MessageSummary) {
	if len(summaries) == 0 {
		fmt.Fprintln(w, "No new messages.")
		return
	}

	fmt.Fprintf(w, "You have %d new message(s):\n", len(summaries))
	for _, msg := range summaries {
		fmt.Fprintf(w, "- %s [%s]: %s\n", msg.Sender, msg.Timestamp.Format(time.RFC3339), msg.Preview)
	}
}
