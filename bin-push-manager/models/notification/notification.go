package notification

// Kind is the kind of a notification.
type Kind string

// list of notification kinds
const (
	KindCallIncoming  Kind = "call_incoming"  // an incoming call to the agent
	KindCallCancelled Kind = "call_cancelled" // the incoming call is over

	KindChatMentioned       Kind = "chat_mentioned"       // the agent is mentioned in a chat message
	KindChatMessage         Kind = "chat_message"         // a message in a chat the agent takes part in
	KindConversationMessage Kind = "conversation_message" // a customer message in a conversation the agent owns
	KindCaseAssigned        Kind = "case_assigned"        // a case is assigned to the agent
	KindCaseNote            Kind = "case_note"            // a note on a case the agent owns
)

// settingsKinds is the kinds an agent can switch off. The call kinds are not part of it: a call is never muted.
var settingsKinds = []Kind{
	KindChatMentioned,
	KindChatMessage,
	KindConversationMessage,
	KindCaseAssigned,
	KindCaseNote,
}

// IsValidKind returns true if the kind is known.
func IsValidKind(k Kind) bool {
	switch k {
	case KindCallIncoming, KindCallCancelled,
		KindChatMentioned, KindChatMessage, KindConversationMessage, KindCaseAssigned, KindCaseNote:
		return true
	}
	return false
}

// SettingsKinds returns the kinds an agent can switch on and off, in a fixed order.
func SettingsKinds() []Kind {
	res := make([]Kind, len(settingsKinds))
	copy(res, settingsKinds)

	return res
}

// IsSettingsKind returns true if an agent can switch the kind on and off.
func IsSettingsKind(k Kind) bool {
	for _, s := range settingsKinds {
		if s == k {
			return true
		}
	}
	return false
}

// DataKeyKind is the data key that push-manager sets itself. A caller must not use it.
const DataKeyKind = "kind"

// SendResult is the result of a notification send request.
type SendResult struct {
	DeviceCount int `json:"device_count"` // number of devices the notification was addressed to
	SentCount   int `json:"sent_count"`   // number of devices that accepted the notification
}
