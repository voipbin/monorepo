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

func (h *messageHandler) StreamingSend(ctx context.Context, aicallID uuid.UUID, role message.Role, content string, returnResponse bool) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "StreamingSend",
		"aicall_id": aicallID,
		"role":      role,
		"content":   content,
	})
	log.Debugf("Sending ai message.")

	cc, err := h.reqHandler.AIV1AIcallGet(ctx, aicallID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the aicall correctly")
	}

	if cc.Status == aicall.StatusEnd {
		return nil, errors.New("aicall is already ended")
	}

	// create a message for outgoing(request)
	res, err := h.Create(ctx, uuid.Nil, cc.CustomerID, aicallID, message.DirectionOutgoing, role, content)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the sending message correctly")
	}

	t1 := time.Now()
	// var tmpMessage *message.Message
	chanMsg := make(<-chan string)

	modelTarget := ai.GetEngineModelTarget(cc.AIEngineModel)
	switch modelTarget {
	case ai.EngineModelTargetOpenai:
		chanMsg, err = h.streamingSendOpenai(ctx, cc)

	default:
		err = fmt.Errorf("unsupported ai engine model: %s", cc.AIEngineModel)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the message correctly")
	}
	// log.Debugf("Received response message from the ai engine. message: %v", tmpMessage)

	t2 := time.Since(t1)
	promMessageProcessTime.WithLabelValues(string(cc.AIEngineType)).Observe(float64(t2.Milliseconds()))

	msgID := h.utilHandler.UUIDCreate()
	if cc.ReferenceType == aicall.ReferenceTypeCall {
		if errSay := h.reqHandler.TTSV1StreamingSay(ctx, cc.TTSStreamingPodID, cc.TTSStreamingID, msgID, ""); errSay != nil {
			return nil, errors.Wrapf(errSay, "could not say the text via tts streaming. tts_streaming_id: %s", cc.TTSStreamingID)
		}
	}

	totalMessage := ""
	for msg := range chanMsg {
		if cc.ReferenceType == aicall.ReferenceTypeCall {
			if errAdd := h.reqHandler.TTSV1StreamingSayAdd(ctx, cc.TTSStreamingPodID, cc.TTSStreamingID, msgID, msg); errAdd != nil {
				return nil, errors.Wrapf(errAdd, "could not add the text via tts streaming. tts_streaming_id: %s", cc.TTSStreamingID)
			}
		}

		totalMessage += msg
	}
	log.Debugf("Finished receiving the streaming message from the ai engine. total_message: %s", totalMessage)

	tmpResponse, err := h.Create(ctx, msgID, cc.CustomerID, cc.ID, message.DirectionIncoming, role, totalMessage)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the recevied message correctly")
	}
	log.WithField("response", tmpResponse).Debugf("Created the response message. message_id: %s", tmpResponse.ID)

	// if len(tmpMessage.Content) == 0 {
	// 	// if the messsage is empty, return the message as it is
	// 	return tmpMessage, nil
	// }

	// create a message for incoming(response)
	// tmpResponse, err := h.Create(ctx, cc.CustomerID, cc.ID, message.DirectionIncoming, tmpMessage.Role, tmpMessage.Content)
	// if err != nil {
	// 	return nil, errors.Wrapf(err, "could not create the recevied message correctly")
	// }

	if returnResponse {
		res = tmpResponse
	}

	// if cc.ReferenceType == aicall.ReferenceTypeConversation {
	// 	// send it to the conversation
	// 	cm, err := h.reqHandler.ConversationV1MessageSend(ctx, cc.ReferenceID, tmpMessage.Content, nil)
	// 	if err != nil {
	// 		return nil, errors.Wrapf(err, "could not send the message to the conversation correctly")
	// 	}
	// 	log.WithField("conversation_message_id", cm.ID).Debugf("Sent the message to the conversation. conversation_id: %s", cc.ReferenceID)
	// }

	return res, nil

}

func (h *messageHandler) streamingSendOpenai(ctx context.Context, cc *aicall.AIcall) (<-chan string, error) {

	switch cc.ReferenceType {
	case aicall.ReferenceTypeCall:
		return h.streamingSendOpenaiReferenceTypeCall(ctx, cc)

	// case aicall.ReferenceTypeConversation:
	// 	return h.sendOpenaiReferenceTypeConversation(ctx, cc)

	// case aicall.ReferenceTypeNone:
	// 	return h.sendOpenaiReferenceTypeNone(ctx, cc)

	default:
		return nil, fmt.Errorf("unsupported reference type: %s", cc.ReferenceType)
	}
}

func (h *messageHandler) streamingSendOpenaiReferenceTypeCall(ctx context.Context, cc *aicall.AIcall) (<-chan string, error) {
	filters := map[string]string{
		"deleted": "false",
	}

	// note: because of chatgpt needs entire message history, we need to send all messages
	messages, err := h.Gets(ctx, cc.ID, 1000, "", filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the messages correctly")
	}

	slices.Reverse(messages)
	res, err := h.engineOpenaiHandler.StreamingSend(ctx, cc, messages)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the message correctly")
	}

	return res, nil
}
