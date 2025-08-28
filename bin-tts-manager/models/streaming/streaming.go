package streaming

import (
	commonidentity "monorepo/bin-common-handler/models/identity"
	"net"
	"sync"

	"github.com/gofrs/uuid"
)

type Streaming struct {
	commonidentity.Identity

	PodID string `json:"pod_id,omitempty"` // id of the pod where the streaming is running

	ReferenceType ReferenceType `json:"reference_type,omitempty"` // Type of the reference (e.g., call, confbridge)
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty"`   // ID of the reference (e.g., call ID, confbridge ID)
	MessageID     uuid.UUID     `json:"message_id,omitempty"`     // ID of the message associated with the streaming

	Language  string    `json:"language,omitempty"` // Language of the streaming
	Gender    Gender    `json:"gender,omitempty"`
	Direction Direction `json:"direction,omitempty"` // Direction of the streaming

	VendorLock   sync.Mutex `json:"-"` // Lock for synchronizing access to VendorConfig
	VendorName   VendorName `json:"-"` // Vendor of the service (e.g., gcp, aws)
	VendorConfig any        `json:"-"`

	ConnAst net.Conn `json:"-"` // Connection to the Asterisk for the streaming
}

// // Direction represents the direction of the streaming in a call.
type Direction string

const (
	DirectionNone     Direction = ""     // direction is not set
	DirectionIncoming Direction = "in"   // inject the streaming direction to the from call to the service. So, the call can not hear the streaming. But the other side can hear the streaming.
	DirectionOutgoing Direction = "out"  // inject the streaming direction to the from service to the call. So, the call can hear the streaming. But the other side can not hear the streaming.
	DirectionBoth     Direction = "both" // inject the streaming direction to both sides of the call. So, both sides can hear the streaming.
)

type Gender string

// list of gender types
const (
	GenderMale    Gender = "male"
	GenderFemale  Gender = "female"
	GenderNeutral Gender = "neutral"
)

type ReferenceType string

const (
	ReferenceTypeNone       ReferenceType = ""           // reference type is not set
	ReferenceTypeCall       ReferenceType = "call"       // reference type is a call
	ReferenceTypeConfbridge ReferenceType = "confbridge" // reference type is a confbridge (conference bridge)
)

type VendorName string

const (
	VendorNameNone       VendorName = ""           // vendor name is not set
	VendorNameElevenlabs VendorName = "elevenlabs" // elevenlabs vendor
)
