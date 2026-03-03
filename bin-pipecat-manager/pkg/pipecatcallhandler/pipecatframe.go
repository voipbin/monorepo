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
	SendData(se *pipecatcall.Session, messageType int, data []byte)
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
	// Fast path: non-blocking send avoids timer allocation in the common case.
	select {
	case <-pc.Ctx.Done():
		return
	case pc.RunnerWebsocketChan <- frame:
		return
	default:
		// Channel full — fall through to timed wait.
	}

	// Slow path: channel was full, wait briefly with a proper timer.
	// Unlike time.After, time.NewTimer + defer Stop releases the timer
	// immediately when the send succeeds, avoiding GC pressure from
	// leaked timers (~50 allocations/sec in the hot audio path).
	timer := time.NewTimer(defaultPushFrameTimeout)
	defer timer.Stop()

	select {
	case <-pc.Ctx.Done():
		return
	case pc.RunnerWebsocketChan <- frame:
		return
	case <-timer.C:
		dropped := pc.DroppedFrames.Add(1)
		if dropped == 1 || dropped%100 == 0 {
			log := logrus.WithField("pipecatcall_id", pc.ID)
			log.Warnf("Audio frame dropped due to channel backpressure. total_dropped: %d", dropped)
		}
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

	h.SendData(se, websocket.BinaryMessage, tmpData)
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

	h.SendData(se, websocket.BinaryMessage, tmpData)
	return nil
}

func (h *pipecatframeHandler) SendData(se *pipecatcall.Session, messageType int, data []byte) {

	tmpData := &pipecatcall.SessionFrame{
		MessageType: messageType,
		Data:        data,
	}

	h.pushFrame(se, tmpData)
}
