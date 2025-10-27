package pipecatcallhandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/models/pipecatframe"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *pipecatcallHandler) pipecatframeSendAudio(pc *pipecatcall.Pipecatcall, packetID uint64, data []byte) error {
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

	h.pipecatFramePush(pc, frame)
	return nil
}

func (h *pipecatcallHandler) pipecatframeSendText(pc *pipecatcall.Pipecatcall, text string) error {
	frame := &pipecatframe.Frame{
		Frame: &pipecatframe.Frame_Text{
			Text: &pipecatframe.TextFrame{
				Text: text,
			},
		},
	}

	h.pipecatFramePush(pc, frame)
	return nil
}

func (h *pipecatcallHandler) pipecatframeSendRTVIText(pc *pipecatcall.Pipecatcall, id string, text string, runImmediately bool, audioResponse bool) error {
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

	h.pipecatFramePush(pc, frame)

	return nil
}

func (h *pipecatcallHandler) pipecatFramePush(pc *pipecatcall.Pipecatcall, frame *pipecatframe.Frame) {
	pc.RunnerWebsocketChan <- frame
}
func (h *pipecatcallHandler) pipecatFrameSendRun(ctx context.Context, pc *pipecatcall.Pipecatcall) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "runnerStartFrameSend",
		"pipecatcall_id": pc.ID,
	})

	for frame := range pc.RunnerWebsocketChan {
		if errSend := h.sendProtobufFrame(pc.RunnerWebsocket, frame); errSend != nil {
			log.Errorf("could not send the frame: %v", errSend)
		}
	}
}
