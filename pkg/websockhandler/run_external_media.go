package websockhandler

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"

	cmexternalmedia "gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
)

// RunExternalMedia creates a new websocket and starts socket message listen.
func (h *websockHandler) RunExternalMedia(ctx context.Context, w http.ResponseWriter, r *http.Request, a *amagent.Agent, callID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "RunExternalMedia",
		"agent": a,
	})

	// create a websock
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Could not create websocket. err: %v", err)
		return err
	}
	defer ws.Close()
	log.Debugf("Created a new websocket correctly.")

	// get the external media info
	filters := map[string]string{
		"reference_id": callID.String(),
	}
	externalMedias, err := h.reqHandler.CallV1ExternalMediaGets(ctx, "", uint64(1), filters)
	if err != nil || len(externalMedias) == 0 {
		log.Errorf("Could not get external medias. err: %v", err)
		return errors.Wrapf(err, "could not get external medias.")
	}

	// select the first external media
	externalMedia := externalMedias[0]
	log.WithField("external_medias", externalMedias).Debugf("Found external medias. external_media: %v", externalMedia)

	// // we are creating a new context and cancel using the http request.
	// // we are expecting when the websocket closed, everything is closed too.
	newCtx, newCancel := context.WithCancel(r.Context())

	go h.runWebsockExternalMedia(newCtx, newCancel, ws, &externalMedia)
	<-newCtx.Done()

	return nil
}

func (h *websockHandler) runWebsockExternalMedia(
	ctx context.Context,
	cancel context.CancelFunc,
	ws *websocket.Conn,
	externalMedia *cmexternalmedia.ExternalMedia,
) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "runWebsockExternalMedia",
		"external_media": externalMedia,
	})
	defer cancel()

	for {
		// receive the raw from the websock
		m, err := h.receiveBinaryFromWebsock(ctx, ws)
		if err != nil {
			log.Infof("Could not receive the message correctly. Assume the websocket has closed. err: %v", err)
			return
		}

		// send the data to the external media socket
		if errSend := h.sendDataToExternalMediaSock(ctx, m, externalMedia); errSend != nil {
			log.WithField("error", errSend).Infof("Could not send the data to the external media socket. err: %v", errSend)
			return
		}
	}
}

// handleMessage handles the message from the websock and do the subscription/unsubscription.
func (h *websockHandler) sendDataToExternalMediaSock(ctx context.Context, data []byte, externalMedia *cmexternalmedia.ExternalMedia) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "sendDataToExternalMediaSock",
		"external_media": externalMedia,
	})
	log.WithField("external_media", externalMedia).Debugf("Sending data to external media socket. reference_id: %s, local_ip: %s, local_port: %d", externalMedia.ReferenceID, externalMedia.LocalIP, externalMedia.LocalPort)

	targetAddress := fmt.Sprintf("%s:%d", externalMedia.LocalIP, externalMedia.LocalPort)
	lenSent := 0
	lenTotal := len(data)
	for {

		if ctx.Err() != nil {
			// the context is over.
			return fmt.Errorf("the context is over")
		}

		if lenSent >= lenTotal {
			// sent all
			break
		}

		// send the data in a udp packet
		conn, err := net.Dial("udp", targetAddress)
		if err != nil {
			log.Errorf("Could not connect to the asterisk. err: %v", err)
			return errors.Wrapf(err, "could not connect to asterisk. err: %v", err)
		}
		defer conn.Close()

		l, err := bufio.NewReader(conn).Read(data)
		if err != nil {
			log.Errorf("Could not send the data. err: %v", err)
			return errors.Wrapf(err, "could not send the data. err: %v", err)
		}
		log.Debugf("Debug. Sent data correctly. len: %d", l)

		lenSent += l
	}

	return nil
}
