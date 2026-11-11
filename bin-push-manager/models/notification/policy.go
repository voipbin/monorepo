package notification

// Policy is how FCM delivers a kind.
type Policy struct {
	// Priority is the Android message priority: HIGH wakes a sleeping device at once, NORMAL waits for the device.
	Priority string

	// TTL is how long FCM keeps a message that can not be delivered, as a duration in seconds.
	TTL string
}

// The delivery values are first values without a measurement behind them; change them here only.
const (
	priorityHigh   = "HIGH"
	priorityNormal = "NORMAL"

	// ttlCall: an incoming call is over after about a minute.
	ttlCall = "60s"

	// ttlEvent: a message or a case change is still useful an hour later.
	ttlEvent = "3600s"
)

// PolicyOf returns the delivery policy of the kind. FCM keeps only 4 different collapse keys per device and
// drops the others without a rule, so no kind uses a collapse key; the app replaces a shown notification by its
// own notification id. A kind without a policy gets the one of a call.
func PolicyOf(k Kind) Policy {
	switch k {
	case KindChatMentioned, KindConversationMessage, KindCaseAssigned:
		return Policy{Priority: priorityHigh, TTL: ttlEvent}
	case KindChatMessage, KindCaseNote:
		return Policy{Priority: priorityNormal, TTL: ttlEvent}
	}

	return Policy{Priority: priorityHigh, TTL: ttlCall}
}
