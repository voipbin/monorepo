package notification

// Kind is the kind of a notification.
type Kind string

// list of notification kinds
const (
	KindCallIncoming  Kind = "call_incoming"  // an incoming call to the agent
	KindCallCancelled Kind = "call_cancelled" // the incoming call is over
)

// IsValidKind returns true if the kind is known.
func IsValidKind(k Kind) bool {
	switch k {
	case KindCallIncoming, KindCallCancelled:
		return true
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
