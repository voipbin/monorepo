package messagehandler

import (
	"context"
	"fmt"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"slices"
	"time"

	cmmessage "monorepo/bin-conversation-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Send sends a message to the ai engine and returns the sent message.
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

	if cc.ReferenceType == aicall.ReferenceTypeConversation {
		// send it to the conversation
		cm, err := h.reqHandler.ConversationV1MessageSend(ctx, cc.ReferenceID, tmpMessage.Content, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "could not send the message to the conversation correctly")
		}
		log.WithField("conversation_message_id", cm.ID).Debugf("Sent the message to the conversation. conversation_id: %s", cc.ReferenceID)
	}

	return res, nil
}

func (h *messageHandler) sendOpenai(ctx context.Context, cc *aicall.AIcall) (*message.Message, error) {

	switch cc.ReferenceType {
	case aicall.ReferenceTypeCall:
		return h.sendOpenaiReferenceTypeCall(ctx, cc)

	case aicall.ReferenceTypeConversation:
		return h.sendOpenaiReferenceTypeConversation(ctx, cc)

	case aicall.ReferenceTypeNone:
		return h.sendOpenaiReferenceTypeNone(ctx, cc)

	default:
		return nil, fmt.Errorf("unsupported reference type: %s", cc.ReferenceType)
	}
}

func (h *messageHandler) sendOpenaiReferenceTypeNone(ctx context.Context, cc *aicall.AIcall) (*message.Message, error) {
	return h.sendOpenaiReferenceTypeCall(ctx, cc)
}

func (h *messageHandler) sendOpenaiReferenceTypeCall(ctx context.Context, cc *aicall.AIcall) (*message.Message, error) {
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

func (h *messageHandler) sendOpenaiReferenceTypeConversation(ctx context.Context, cc *aicall.AIcall) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "sendOpenaiReferenceTypeConversation",
		"aicall_id": cc.ID,
	})

	// get ai engine
	ai, err := h.reqHandler.AIV1AIGet(ctx, cc.AIID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the ai engine correctly. ai_id: %s", cc.AIID)
	}
	log.WithField("ai_engine", ai).Debugf("Found the ai engine.")

	filters := map[string]string{
		"deleted":         "false",
		"conversation_id": cc.ReferenceID.String(),
	}
	cms, err := h.reqHandler.ConversationV1MessageGets(ctx, "", 100, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the messages correctly")
	}

	// convert the conversation messages into messages
	messages := []*message.Message{}
	for _, cm := range cms {
		role := message.RoleAssistant
		if cm.Direction == cmmessage.DirectionIncoming {
			role = message.RoleUser
		}

		direction := message.DirectionOutgoing
		if cm.Direction == cmmessage.DirectionIncoming {
			direction = message.DirectionIncoming
		}

		m := &message.Message{
			Direction: direction,
			Role:      role,
			Content:   cm.Text,

			TMCreate: cm.TMCreate,
		}

		messages = append(messages, m)
	}
	messages = append(messages, &message.Message{
		Role:    message.RoleSystem,
		Content: ai.InitPrompt,
	})

	slices.Reverse(messages)
	log.WithField("messages", messages).Debugf("Found the messages.")

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
