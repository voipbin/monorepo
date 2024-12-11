package stream

import (
	"net"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

type Stream struct {
	ID              uuid.UUID
	ConnWebsocket   *websocket.Conn
	ConnAusiosocket net.Conn

	Encapsulation Encapsulation
	ExternalMedia *cmexternalmedia.ExternalMedia
}

type Encapsulation string

const (
	EncapsulationNone        Encapsulation = ""
	EncapsulationAudiosocket Encapsulation = "audiosocket"
	EncapsulationRTP         Encapsulation = "rtp"
	EncapsulationSLN         Encapsulation = "sln"
)
