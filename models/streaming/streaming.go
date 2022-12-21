package streaming

import (
	"net"

	"github.com/gofrs/uuid"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/common"
)

// Streaming defines current streaming detail
type Streaming struct {
	ID           uuid.UUID                                `json:"id"`
	TranscribeID uuid.UUID                                `json:"transcribe_id"`
	CustomerID   uuid.UUID                                `json:"customer_id"`
	Language     string                                   `json:"language"`
	Direction    common.Direction                         `json:"direction"`
	Conn         *net.UDPConn                             `json:"-"`
	Stream       speechpb.Speech_StreamingRecognizeClient `json:"-"`
}
