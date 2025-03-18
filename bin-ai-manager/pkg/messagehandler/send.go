package messagehandler

import (
	"context"
	"fmt"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"slices"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *messageHandler) Send(ctx context.Context, aicallID uuid.UUID, role message.Role, content string) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "Send",
		"aicall_id": aicallID,
		"role":      role,
		"content":   content,
	})
	log.Debugf("Sending ai message.")

	cc, err := h.aicallHandler.Get(ctx, aicallID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the aicall correctly")
	}

	if cc.Status == aicall.StatusEnd {
		return nil, errors.New("aicall is already ended")
	}

	// create a message for outgoing(request)
	res, err := h.Create(ctx, cc.CustomerID, aicallID, message.DirectionOutgoing, role, content)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the sending message correctly")
	}

	t1 := time.Now()
	var tmpMessage *message.Message

	modelTarget := ai.GetEngineModelTarget(cc.AIEngineModel)
	switch modelTarget {
	case ai.EngineModelTargetOpenai:
		tmpMessage, err = h.sendOpenai(ctx, cc)

	case ai.EngineModelTargetDialogflow:
		tmpMessage, err = h.sendDialogflow(ctx, cc, res)

	default:
		err = fmt.Errorf("unsupported ai engine model: %s", cc.AIEngineModel)

	}
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the message correctly")
	}
	log.Debugf("Received response message from the ai engine. message: %v", tmpMessage)

	t2 := time.Since(t1)
	promMessageProcessTime.WithLabelValues(string(cc.AIEngineType)).Observe(float64(t2.Milliseconds()))

	if len(tmpMessage.Content) == 0 {
		// if the messsage is empty, return the message as it is
		return tmpMessage, nil
	}

	// create a message for incoming(response)
	_, err = h.Create(ctx, cc.CustomerID, cc.ID, message.DirectionIncoming, tmpMessage.Role, tmpMessage.Content)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the recevied message correctly")
	}

	return res, nil
}

func (h *messageHandler) sendOpenai(ctx context.Context, cc *aicall.AIcall) (*message.Message, error) {
	filters := map[string]string{
		"deleted": "false",
	}

	// note: because of chatgpt needs entire message history, we need to send all messages
	messages, err := h.Gets(ctx, cc.ID, 1000, "", filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the messages correctly")
	}

	slices.Reverse(messages)
	res, err := h.engineOpenaiHandler.MessageSend(ctx, cc, messages)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the message correctly")
	}

	return res, nil
}

func (h *messageHandler) sendDialogflow(ctx context.Context, cc *aicall.AIcall, m *message.Message) (*message.Message, error) {
	res, err := h.engineDialogflowHandler.MessageSend(ctx, cc, m)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the message correctly")
	}

	return res, nil
}
