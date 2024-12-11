package streamhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/models/stream"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
)

func (h *streamHandler) Start(
	ctx context.Context,
	ws *websocket.Conn,
	referenceType cmexternalmedia.ReferenceType,
	referenceID uuid.UUID,
	encapsulation stream.Encapsulation,
) (*stream.Stream, error) {
	log := logrus.WithField("func", "Start")

	id := h.utilHandler.UUIDCreate()
	log = log.WithField("id", id)

	// create stream should be done before starting stream handler because
	// the call-manager(asterisk) will initiate the stream sending immediately.
	//
	// create stream
	tmp, err := h.Create(ctx, id, ws, encapsulation)
	if err != nil {
		log.Errorf("Could not create stream. err: %v", err)
		return nil, fmt.Errorf("could not create stream. err: %v", err)
	}

	em, err := h.reqHandler.CallV1ExternalMediaStart(
		ctx,
		tmp.ID,
		referenceType,
		referenceID,
		false,
		h.listenAddress,
		defaultExternalMediaEncapsulation,
		defaultExternalMediaTransport,
		defaultExternalMediaConnectionType,
		defaultExternalMediaFormat,
		defaultExternalMediaDirection,
	)
	if err != nil {
		log.Errorf("Could not start the external media. err: %v", err)
		return nil, fmt.Errorf("could not start the external media. err: %v", err)
	}
	log.WithField("external_media", em).Debugf("Created external media. external_media_id: %s", em.ID)

	res, err := h.SetExternalMedia(tmp.ID, em)
	if err != nil {
		log.Errorf("Could not set external media. err: %v", err)
		return nil, fmt.Errorf("could not set external media. err: %v", err)
	}

	go h.handleStreamFromWebsocket(ctx, res)

	return res, nil
}
