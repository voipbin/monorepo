package convtitle

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-conversation-manager/models/conversation"
)

const titleSep = " · " // U+00B7 MIDDLE DOT

// Build returns the auto-generated name and detail for a new conversation.
func Build(convType conversation.Type, peer commonaddress.Address) (name, detail string) {
	label := channelLabel(convType)
	name = label + titleSep + peerName(peer)
	detail = label + " conversation"
	return
}

// channelLabel returns the human-readable channel name for a conversation type.
// When adding a new conversation.Type, add a case here — do not rely on the fallback.
func channelLabel(t conversation.Type) string {
	switch t {
	case conversation.TypeLine:
		return "LINE"
	case conversation.TypeMessage:
		return "SMS"
	default:
		return string(t)
	}
}

// peerName returns the best available display name for a peer address.
// For human-readable address types (tel, email, sip, extension), the raw
// Target is appended in parentheses when a TargetName is also present.
// For opaque types (line user IDs, UUIDs), the raw Target is suppressed.
func peerName(peer commonaddress.Address) string {
	if peer.TargetName != "" {
		if humanReadableTarget(peer.Type) && peer.Target != "" {
			return peer.TargetName + " (" + peer.Target + ")"
		}
		return peer.TargetName
	}
	if peer.Target != "" {
		return peer.Target
	}
	return "Unknown"
}

// humanReadableTarget returns true when the address Target field contains
// a human-readable identifier (phone number, email, SIP URI, extension).
// New address types with human-readable targets must be added here explicitly.
// Unknown types default to false (opaque) for safety.
func humanReadableTarget(t commonaddress.Type) bool {
	switch t {
	case commonaddress.TypeTel, commonaddress.TypeEmail,
		commonaddress.TypeSIP, commonaddress.TypeExtension:
		return true
	default:
		return false
	}
}
