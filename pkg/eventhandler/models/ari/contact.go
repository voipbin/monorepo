package ari

// ContactStatusType type
type ContactStatusType string

// ContactStatusType list
const (
	ContactStatusTypeUnreachable  ContactStatusType = "Unreachable"
	ContactStatusTypeReachable    ContactStatusType = "Reachable"
	ContactStatusTypeUnknown      ContactStatusType = "Unknown"
	ContactStatusTypeNonQualified ContactStatusType = "NonQualified"
	ContactStatusTypeRemoved      ContactStatusType = "Removed"
)

// ContactInfo struct
type ContactInfo struct {
	AOR           string            `json:"aor"`
	URI           string            `json:"uri"`
	RoundtripUsec string            `json:"roundtrip_usec"`
	ContactStatus ContactStatusType `json:"contact_status"`
}

// ContactStatusChange struct for ContactStatusChange ARI event
type ContactStatusChange struct {
	Event
	Endpoint    Endpoint    `json:"endpoint"`
	ContactInfo ContactInfo `json:"contact_info"`
}
