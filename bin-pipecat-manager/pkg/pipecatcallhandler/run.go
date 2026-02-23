package pipecatcallhandler

import (
	"context"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	defaultMediaSampleRate = 16000
	defaultMediaNumChannel = 1
)

func (h *pipecatcallHandler) runAsteriskReceivedMediaHandle(se *pipecatcall.Session) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "runAsteriskReceivedMediaHandle",
		"pipecatcall_id": se.ID,
	})

	if se.ConnAst == nil {
		log.Debugf("No Asterisk WebSocket connection, skipping media handle.")
		return
	}

	packetID := uint64(0)
	for {
		if se.Ctx.Err() != nil {
			log.Debugf("Context has finished. pipecatcall_id: %s", se.ID)
			return
		}

		msgType, data, err := h.websocketHandler.ReadMessage(se.ConnAst)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Debugf("Asterisk WebSocket closed normally.")
			} else {
				log.Infof("Asterisk WebSocket read error: %v", err)
			}
			return
		}

		if msgType != websocket.BinaryMessage {
			continue
		}

		if len(data) == 0 {
			continue
		}

		if errSend := h.pipecatframeHandler.SendAudio(se, packetID, data); errSend != nil {
			log.Errorf("Could not send audio frame. err: %v", errSend)
		}

		packetID++
	}
}

func (h *pipecatcallHandler) runGetLLMKey(ctx context.Context, pc *pipecatcall.Pipecatcall) string {
	switch pc.ReferenceType {
	case pipecatcall.ReferenceTypeAICall:
		c, err := h.requestHandler.AIV1AIcallGet(ctx, pc.ReferenceID)
		if err != nil {
			logrus.Errorf("Could not get ai call info. err: %v", err)
			return ""
		}

		a, err := h.requestHandler.AIV1AIGet(ctx, c.AIID)
		if err != nil {
			logrus.Errorf("Could not get ai info. err: %v", err)
			return ""
		}

		return a.EngineKey

	default:
		return ""
	}
}
