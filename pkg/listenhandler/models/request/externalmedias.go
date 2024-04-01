package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
)

// V1DataExternalMediasPost is
// v1 data type request struct for
// /v1/external-medias POST
type V1DataExternalMediasPost struct {
	ReferenceType  externalmedia.ReferenceType `json:"reference_type,omitempty"`
	ReferenceID    uuid.UUID                   `json:"reference_id,omitempty"`
	NoInsertMedia  bool                        `json:"no_insert_media,omitempty"` // note: do not set this true without caution. the only transcribe-manager sets this to true.
	ExternalHost   string                      `json:"external_host,omitempty"`
	Encapsulation  string                      `json:"encapsulation,omitempty"`
	Transport      string                      `json:"transport,omitempty"`
	ConnectionType string                      `json:"connection_type,omitempty"`
	Format         string                      `json:"format,omitempty"`
	Direction      string                      `json:"direction,omitempty"` // in, out, both
}
