package address

// Address contains source or destination detail info.
type Address struct {
	Type       Type   `json:"type,omitempty"`        // type of address
	Target     string `json:"target,omitempty"`      // address endpoint
	TargetName string `json:"target_name,omitempty"` // address's name.
	Name       string `json:"name,omitempty"`        // name
	Detail     string `json:"detail,omitempty"`      // detail description.
}

// Type define
type Type string

// List of Types
const (
	TypeNone       Type = ""            // no type specified
	TypeAgent      Type = "agent"       // target is agent's id.
	TypeAI         Type = "ai"          // target is AI resource's id
	TypeAITeam     Type = "ai_team"     // target is AI team's id
	TypeConference Type = "conference"  // target is conference's id
	TypeEmail      Type = "email"       // target is email address
	TypeExtension  Type = "extension"   // target is extension
	TypeLine       Type = "line"        // target is naver line's id
	TypeSIP        Type = "sip"         // target is sip destination
	TypeTel        Type = "tel"         // target tel number
	TypeWebchat    Type = "webchat"     // target is webchat-manager's Session.ID or Widget.ID
	TypeWebSession Type = "web_session" // target is webchat-manager's Session.ID (the visitor's continuity token)
	TypeWhatsApp   Type = "whatsapp"    // target is WhatsApp phone number
)

// IsExternalEndpoint reports whether a target of type t is a value the user
// supplies (a phone number, a SIP URI, an email address, ...) and not the id
// of a platform resource (an agent, an AI, a conference, ...).
//
// It is an allow-list on purpose: a type added later is treated as a
// resource reference until someone decides otherwise. The Flow AI Builder
// uses it to keep a model from putting an arbitrary id into an address
// (VOIP-1573): a resource target is cleared and the user picks it in the
// editor.
func IsExternalEndpoint(t Type) bool {
	switch t {
	case TypeTel, TypeSIP, TypeEmail, TypeLine, TypeWhatsApp:
		return true
	}
	return false
}
