package pipecatcallhandler

import (
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/models/pipecatframe"

	"github.com/pkg/errors"
)

func (h *pipecatcallHandler) pipecatframeSendAudio(pc *pipecatcall.Pipecatcall, packetID uint64, data []byte) error {
	pipecatFrame := &pipecatframe.Frame{
		Frame: &pipecatframe.Frame_Audio{
			Audio: &pipecatframe.AudioRawFrame{
				Id:          packetID,
				Audio:       data,
				SampleRate:  defaultMediaSampleRate,
				NumChannels: defaultMediaNumChannel,
			},
		},
	}

	if pc.RunnerWebsocket != nil {
		if errSend := h.sendProtobufFrame(pc.RunnerWebsocket, pipecatFrame); errSend != nil {
			return errors.Wrapf(errSend, "could not send the frame")
		}
	}

	return nil
}

func (h *pipecatcallHandler) pipecatframeSendText(pc *pipecatcall.Pipecatcall, text string) error {
	pipecatFrame := &pipecatframe.Frame{
		Frame: &pipecatframe.Frame_Text{
			Text: &pipecatframe.TextFrame{
				Text: text,
			},
		},
	}

	if pc.RunnerWebsocket != nil {
		if errSend := h.sendProtobufFrame(pc.RunnerWebsocket, pipecatFrame); errSend != nil {
			return errors.Wrapf(errSend, "could not send the frame")
		}
	}

	return nil
}
