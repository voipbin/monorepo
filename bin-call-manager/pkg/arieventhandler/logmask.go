package arieventhandler

import (
	"regexp"
)

// pushRingTicketRegex matches the one-time join ticket of a push ring session as it arrives in the SIP request:
// "direct." and 32 lowercase hex characters. The direct hashes of the direct manager are shorter and do not match.
var pushRingTicketRegex = regexp.MustCompile(`direct\.[0-9a-f]{32}`)

// maskPushRingTicket masks the push ring join ticket in the given value so that it does not reach the logs. A ticket
// shows its first 4 hex characters only. A value without a ticket is returned as it is.
func maskPushRingTicket(v string) string {
	return pushRingTicketRegex.ReplaceAllStringFunc(v, func(m string) string {
		const prefix = len("direct.")
		return m[:prefix+4] + "****"
	})
}
