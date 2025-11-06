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
	RunSender(se *pipecatcall.Session)

	SendAudio(se *pipecatcall.Session, packetID uint64, data []byte) error
	SendRTVIText(se *pipecatcall.Session, id string, text string, runImmediately bool, audioResponse bool) error
}

type pipecatframeHandler struct {
	websocketHandler WebsocketHandler
}

func NewPipecatframeHandler() *pipecatframeHandler {
	return &pipecatframeHandler{
		websocketHandler: NewWebsocketHandler(),
	}
}

func (h *pipecatframeHandler) RunSender(se *pipecatcall.Session) {
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

			if errSend := h.sendFrame(se.RunnerWebsocketWrite, frame); errSend != nil {
				log.Errorf("Could not send the frame. Stopping sender runner. err: %v", errSend)
				return
			}
		}
	}

}

func (h *pipecatframeHandler) sendFrame(conn *websocket.Conn, frame *pipecatframe.Frame) error {
	if conn == nil {
		// connection is not ready. drop the frame
		return nil
	}

	marshaledFrame, err := proto.Marshal(frame)
	if err != nil {
		return errors.Wrapf(err, "could not marshal the frame")
	}

	if errSend := h.websocketHandler.WriteMessage(conn, websocket.BinaryMessage, marshaledFrame); errSend != nil {
		return errors.Wrapf(errSend, "could not send the frame")
	}

	return nil
}

func (h *pipecatframeHandler) pushFrame(pc *pipecatcall.Session, frame *pipecatframe.Frame) {
	select {
	case <-pc.Ctx.Done():
		return

	case pc.RunnerWebsocketChan <- frame:
		return

	case <-time.After(defaultPushFrameTimeout):
		logrus.WithFields(logrus.Fields{
			"func":           "pushFrame",
			"pipecatcall_id": pc.ID,
		}).Warnf("Timeout pushing frame to RunnerWebsocketChan, dropping frame")
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

	h.pushFrame(se, frame)
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

	h.pushFrame(se, frame)

	return nil
}
