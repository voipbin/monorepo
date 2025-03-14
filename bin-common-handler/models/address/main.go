package address

// Address contains source or destination detail info.
type Address struct {
	Type       Type   `json:"type"`        // type of address
	Target     string `json:"target"`      // address endpoint
	TargetName string `json:"target_name"` // address's name.
	Name       string `json:"name"`        // name
	Detail     string `json:"detail"`      // detail description.
}

// Type define
type Type string

// List of Types
const (
	TypeNone       Type = ""           // no type specified
	TypeAgent      Type = "agent"      // target is agent's id.
	TypeConference Type = "conference" // target is conference's id
	TypeEmail      Type = "email"      // target is email address
	TypeExtension  Type = "extension"  // target is extension
	TypeLine       Type = "line"       // target is naver line's id
	TypeSIP        Type = "sip"        // target is sip destination
	TypeTel        Type = "tel"        // target tel number
)
