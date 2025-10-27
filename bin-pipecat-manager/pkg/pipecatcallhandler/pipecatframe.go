package pipecatcallhandler

import (
	"encoding/json"
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
		pc.RunnerWebsocketMu.Lock()
		defer pc.RunnerWebsocketMu.Unlock()

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
		pc.RunnerWebsocketMu.Lock()
		defer pc.RunnerWebsocketMu.Unlock()

		if errSend := h.sendProtobufFrame(pc.RunnerWebsocket, pipecatFrame); errSend != nil {
			return errors.Wrapf(errSend, "could not send the frame")
		}
	}

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

	pipecatFrame := &pipecatframe.Frame{
		Frame: &pipecatframe.Frame_Message{
			Message: &pipecatframe.MessageFrame{
				Data: string(data),
			},
		},
	}

	if pc.RunnerWebsocket != nil {
		pc.RunnerWebsocketMu.Lock()
		defer pc.RunnerWebsocketMu.Unlock()

		if errSend := h.sendProtobufFrame(pc.RunnerWebsocket, pipecatFrame); errSend != nil {
			return errors.Wrapf(errSend, "could not send the frame")
		}
	}

	return nil
}
