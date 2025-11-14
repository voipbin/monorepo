package pipecatcallhandler

//go:generate mockgen -package pipecatcallhandler -destination ./mock_pipecatframe.go -source pipecatframe.go -build_flags=-mod=mod

import (
	"encoding/json"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/models/pipecatframe"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

type PipecatframeHandler interface {
	RunSender(se *pipecatcall.Session, ws *websocket.Conn)

	SendAudio(se *pipecatcall.Session, packetID uint64, data []byte) error
	SendRTVIText(se *pipecatcall.Session, id string, text string, runImmediately bool, audioResponse bool) error
	SendDataRaw(se *pipecatcall.Session, messageType int, data []byte)
}

type pipecatframeHandler struct {
	websocketHandler WebsocketHandler
}

func NewPipecatframeHandler() *pipecatframeHandler {
	return &pipecatframeHandler{
		websocketHandler: NewWebsocketHandler(),
	}
}

func (h *pipecatframeHandler) RunSender(se *pipecatcall.Session, ws *websocket.Conn) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "RunSender",
		"pipecatcall_id": se.ID,
	})

	for {
		select {
		case <-se.Ctx.Done():
			log.Infof("Context done, stopping sender runner.")
			return

		case frame, ok := <-se.RunnerWebsocketChan:
			if !ok {
				log.Infof("RunnerWebsocketChan closed, stopping sender runner.")
				return
			}

			if frame == nil {
				log.Warnf("Received nil frame, skipping.")
				continue
			}

			if errSend := h.websocketHandler.WriteMessage(ws, frame.MessageType, frame.Data); errSend != nil {
				log.Errorf("Could not send the frame. Stopping sender runner. err: %v", errSend)
				return
			}
		}
	}

}

func (h *pipecatframeHandler) pushFrame(pc *pipecatcall.Session, frame *pipecatcall.SessionFrame) {
	select {
	case <-pc.Ctx.Done():
		return

	case pc.RunnerWebsocketChan <- frame:
		return

	case <-time.After(defaultPushFrameTimeout):
		// timeout pushing frame
		// note: we don't log here to avoid flooding the logs
		// just drop the frame
		return
	}
}

func (h *pipecatframeHandler) SendAudio(se *pipecatcall.Session, packetID uint64, data []byte) error {
	frame := &pipecatframe.Frame{
		Frame: &pipecatframe.Frame_Audio{
			Audio: &pipecatframe.AudioRawFrame{
				Id:          packetID,
				Audio:       data,
				SampleRate:  defaultMediaSampleRate,
				NumChannels: defaultMediaNumChannel,
			},
		},
	}

	tmpData, err := proto.Marshal(frame)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the frame")
	}

	h.SendDataRaw(se, websocket.BinaryMessage, tmpData)
	return nil
}

func (h *pipecatframeHandler) SendRTVIText(se *pipecatcall.Session, id string, text string, runImmediately bool, audioResponse bool) error {
	tmp := pipecatframe.CommonFrameMessage{
		ID:    id,
		Label: pipecatframe.RTVIMessageLabel,
		Type:  pipecatframe.RTVIFrameTypeSendText,
		Data: pipecatframe.RTVISendTextData{
			Content: text,
			Options: &pipecatframe.RTVISendTextOptions{
				RunImmediately: runImmediately,
				AudioResponse:  audioResponse,
			},
		},
	}

	data, err := json.Marshal(&tmp)
	if err != nil {
		return errors.Wrapf(err, "could not marshal RTVISendTextData")
	}

	frame := &pipecatframe.Frame{
		Frame: &pipecatframe.Frame_Message{
			Message: &pipecatframe.MessageFrame{
				Data: string(data),
			},
		},
	}

	tmpData, err := proto.Marshal(frame)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the frame")
	}

	h.SendDataRaw(se, websocket.BinaryMessage, tmpData)
	return nil
}

func (h *pipecatframeHandler) SendDataRaw(se *pipecatcall.Session, messageType int, data []byte) {

	tmpData := &pipecatcall.SessionFrame{
		MessageType: messageType,
		Data:        data,
	}

	h.pushFrame(se, tmpData)
}
