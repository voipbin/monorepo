package streaming

import (
	"net"

	speechpb "cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
)

// Streaming defines current streaming detail
type Streaming struct {
	ID              uuid.UUID                                `json:"id"`
	CustomerID      uuid.UUID                                `json:"customer_id"`
	TranscribeID    uuid.UUID                                `json:"transcribe_id"`
	ExternalMediaID uuid.UUID                                `json:"external_media_id"`
	Language        string                                   `json:"language"`
	Direction       transcript.Direction                     `json:"direction"`
	Conn            *net.UDPConn                             `json:"-"`
	Stream          speechpb.Speech_StreamingRecognizeClient `json:"-"`
}
