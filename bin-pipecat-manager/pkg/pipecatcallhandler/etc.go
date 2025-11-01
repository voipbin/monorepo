package pipecatcallhandler

import (
	"context"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *pipecatcallHandler) SendMessage(ctx context.Context, id uuid.UUID, messageID string, messageText string, runImmediately bool, audioResponse bool) (*message.Message, error) {
	se, err := h.SessionGet(id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pipecatcall info")
	}

	tmpID := h.utilHandler.UUIDCreate()
	res := message.Message{
		Identity: commonidentity.Identity{
			ID:         tmpID,
			CustomerID: se.CustomerID,
		},

		PipecatcallID:            se.ID,
		PipecatcallReferenceType: se.PipecatcallReferenceType,
		PipecatcallReferenceID:   se.PipecatcallReferenceID,

		Text: messageText,
	}
	// h.notifyHandler.PublishEvent(ctx, message.EventTypeUserTranscription, res)

	if errSend := h.pipecatframeHandler.SendRTVIText(se, messageID, messageText, runImmediately, audioResponse); errSend != nil {
		return nil, errors.Wrapf(errSend, "could not send the message to pipecatcall")
	}

	return &res, nil
}
