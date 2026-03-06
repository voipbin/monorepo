package pipecatcallhandler

import (
	"context"
	"monorepo/bin-pipecat-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *pipecatcallHandler) SendMessage(ctx context.Context, id uuid.UUID, messageID string, messageText string, runImmediately bool, audioResponse bool) (*message.Message, error) {
	se, err := h.SessionGet(id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pipecatcall info")
	}

	res := h.newMessageEvent(se, messageText)

	if errSend := h.pipecatframeHandler.SendRTVIText(se, messageID, messageText, runImmediately, audioResponse); errSend != nil {
		return nil, errors.Wrapf(errSend, "could not send the message to pipecatcall")
	}

	return &res, nil
}
