package hook

// Hook Defines
type Hook struct {
	Type   Type     `json:"type"`
	Topics []string `json:"topics"`
}

// Type defines hook type
type Type string

// list of types
const (
	TypeSubscribe   Type = "subscribe"
	TypeUnsubscribe Type = "unsubscribe"
)
