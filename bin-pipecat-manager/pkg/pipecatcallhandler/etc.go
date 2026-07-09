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

	// Record the correlation for the upcoming LLM generation (VOIP-1234 §4-1).
	// If messageID is not a valid UUID (unexpected but non-fatal — this is a
	// best-effort correlation hint, not load-bearing for delivery), leave
	// pendingInReplyToMessageID at its previous value rather than failing the
	// send; the runner will snapshot whatever is there when the generation
	// starts. Goes through Session's exported setter (not a direct field
	// write) because SendMessage runs on the RPC worker pool goroutine while
	// the WebSocket read loop concurrently reads this value.
	if parsed, errParse := uuid.FromString(messageID); errParse == nil {
		se.SetPendingInReplyToMessageID(parsed)
	}

	res := h.newMessageEvent(se, messageText)

	if errSend := h.pipecatframeHandler.SendRTVIText(se, messageID, messageText, runImmediately, audioResponse); errSend != nil {
		return nil, errors.Wrapf(errSend, "could not send the message to pipecatcall")
	}

	return &res, nil
}
