package confbridge

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Confbridge define
type Confbridge struct {
	commonidentity.Identity

	ActiveflowID  uuid.UUID     `json:"activeflow_id,omitempty"`  // activeflow id
	ReferenceType ReferenceType `json:"reference_type,omitempty"` // reference type
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty"`   // reference id

	Type     Type   `json:"type"`
	Status   Status `json:"status"`
	BridgeID string `json:"bridge_id"` // bridge id
	Flags    []Flag `json:"flags"`     // list of flags

	ChannelCallIDs map[string]uuid.UUID `json:"channel_call_ids"` // channelid:callid

	// recording id for currently recording this confbridge. if recording is ongoing, the recording request will be rejected.
	// but this is optional. by the recording handler, it is possible to recording the confbridge without this limitation and
	// it will not set the recording id here.
	RecordingID  uuid.UUID   `json:"recording_id"`
	RecordingIDs []uuid.UUID `json:"recording_ids"` // list of recording ids.

	// current external media id
	ExternalMediaID uuid.UUID `json:"external_media_id"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

type ReferenceType string

const (
	ReferenceTypeCall       ReferenceType = "call"
	ReferenceTypeConference ReferenceType = "conference"
	ReferenceTypeAI         ReferenceType = "ai"
	ReferenceTypeQueue      ReferenceType = "queue"
	ReferenceTranscribe     ReferenceType = "transcribe"
	ReferenceTransfer       ReferenceType = "transfer"
)

// Type define
type Type string

// list of types
const (
	TypeConnect    Type = "connect"    // the confbridge will be terminated if there is only 1 channel left in the bridge.
	TypeConference Type = "conference" // the confbridge will not be terminated until someone sends a request to destroy it.
)

// Status define
type Status string

// list of statuses
const (
	StatusProgressing Status = "progressing" // confbridge is ongoing
	StatusTerminating Status = "terminating" // confbridge is terminating
	StatusTerminated  Status = "terminated"  // confbridge terminated.
)

// Flag define
type Flag string

// list of flags
const (
	FlagNoAutoLeave Flag = "no_auto_leave" // blocks auto leave for connect type of confbridge. if this sets, it blocks the last call leave in the confbridge.
)
