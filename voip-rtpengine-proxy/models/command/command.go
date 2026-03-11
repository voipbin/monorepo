package command

// Type represents the command type for rtpengine-proxy requests.
type Type string

const (
	// TypeNG routes commands to the RTPEngine NG protocol client.
	TypeNG Type = "ng"

	// TypeExec starts a tcpdump process via the process manager.
	TypeExec Type = "exec"

	// TypeKill stops a running tcpdump process via the process manager.
	TypeKill Type = "kill"
)

// Command represents an incoming command request to rtpengine-proxy.
// Typed fields are extracted for routing and validation.
// Data holds the full raw payload for NG protocol forwarding.
type Command struct {
	Type       Type
	ID         string
	Command    string
	Parameters []string
	Data       map[string]interface{}
}
