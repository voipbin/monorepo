package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
)

// V1DataExternalMediasPost is
// v1 data type request struct for
// /v1/external-medias POST
type V1DataExternalMediasPost struct {
	ReferenceType  externalmedia.ReferenceType `json:"reference_type"`
	ReferenceID    uuid.UUID                   `json:"reference_id"`
	ExternalHost   string                      `json:"external_host"`
	Encapsulation  string                      `json:"encapsulation"`
	Transport      string                      `json:"transport"`
	ConnectionType string                      `json:"connection_type"`
	Format         string                      `json:"format"`
	Direction      string                      `json:"direction"` // in, out
}
