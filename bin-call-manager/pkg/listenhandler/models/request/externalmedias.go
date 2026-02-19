package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/externalmedia"
)

// V1DataExternalMediasPost is
// v1 data type request struct for
// /v1/external-medias POST
type V1DataExternalMediasPost struct {
	ID            uuid.UUID                   `json:"id,omitempty"`
	Type          externalmedia.Type          `json:"type,omitempty"`
	ReferenceType externalmedia.ReferenceType `json:"reference_type,omitempty"`
	ReferenceID     uuid.UUID                   `json:"reference_id,omitempty"`
	ExternalHost    string                      `json:"external_host,omitempty"`
	Encapsulation   string                      `json:"encapsulation,omitempty"`
	Transport       string                      `json:"transport,omitempty"`
	ConnectionType  string                      `json:"connection_type,omitempty"`
	Format          string                      `json:"format,omitempty"`
	Direction       string                      `json:"direction,omitempty"`        // in, out, both
	DirectionListen externalmedia.Direction     `json:"direction_listen,omitempty"` // in, out, both, default is empty
	DirectionSpeak  externalmedia.Direction     `json:"direction_speak,omitempty"`  // in, out, both, default is empty
}
