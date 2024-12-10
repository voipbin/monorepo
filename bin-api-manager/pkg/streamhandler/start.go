package streamhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
)

func (h *streamHandler) Start(ctx context.Context, ws *websocket.Conn, referenceType cmexternalmedia.ReferenceType, referenceID uuid.UUID, localEndpoint string) error {
	// generate external media id
	externalMediaID := h.utilHandler.UUIDCreate()

	em, err := h.reqHandler.CallV1ExternalMediaStart(
		ctx,
		uuid.Nil,
		referenceType,
		referenceID,
		false,
		h.listenAddress,
		defaultEncapsulationForAudioSocket,
		defaultTransportForAudioSocket,
		defaultConnectionType,
		defaultFormat,
		defualtDirection,
	)

}
