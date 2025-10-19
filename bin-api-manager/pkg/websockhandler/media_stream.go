package websockhandler

import (
	"context"
	"net/http"

	"monorepo/bin-api-manager/models/stream"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// mediaStreamRun starts the media stream forwarding
func (h *websockHandler) mediaStreamRun(ctx context.Context, w http.ResponseWriter, r *http.Request, referenceType cmexternalmedia.ReferenceType, referenceID uuid.UUID, encapsulation string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "mediaStreamRun",
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"encapsulation":  encapsulation,
	})

	// create a websock
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Could not create websocket. err: %v", err)
		return err
	}
	defer func() {
		_ = ws.Close()
	}()
	log.Debugf("Created a new websocket correctly.")

	st, err := h.streamHandler.Start(ctx, ws, referenceType, referenceID, stream.Encapsulation(encapsulation))
	if err != nil {
		log.Errorf("Could not start the stream handler. err: %v", err)
		return err
	}
	log.WithField("stream", st).Debugf("The stream handler started. stream_id: %s", st.ID)

	<-ctx.Done()

	return nil
}
