package messagehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	fmaction "monorepo/bin-flow-manager/models/action"
	"sync"

	"slices"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *messageHandler) StreamingSend(ctx context.Context, aicallID uuid.UUID, role message.Role, content string) (*message.Message, error) {
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

	if cc.Status == aicall.StatusTerminated {
		return nil, errors.New("aicall is already ended")
	} else if cc.ReferenceType != aicall.ReferenceTypeCall {
		return nil, fmt.Errorf("unsupported reference type: %s", cc.ReferenceType)
	}

	// create a message for outgoing(request)
	res, err := h.Create(ctx, uuid.Nil, cc.CustomerID, aicallID, message.DirectionOutgoing, role, content)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the sending message correctly")
	}

	t1 := time.Now()
	var chanText <-chan string
	var chanTool <-chan *fmaction.Action

	modelTarget := ai.GetEngineModelTarget(cc.AIEngineModel)
	switch modelTarget {
	case ai.EngineModelTargetOpenai:
		chanText, chanTool, err = h.streamingSendOpenai(ctx, cc)

	default:
		err = fmt.Errorf("unsupported ai engine model: %s", cc.AIEngineModel)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the message correctly")
	}

	t2 := time.Since(t1)
	promMessageProcessTime.WithLabelValues(string(cc.AIEngineType)).Observe(float64(t2.Milliseconds()))

	msgID := h.utilHandler.UUIDCreate()
	tmp, err := h.reqHandler.TTSV1StreamingSayInit(ctx, cc.TTSStreamingPodID, cc.TTSStreamingID, msgID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not say the text via tts streaming. tts_streaming_id: %s", cc.TTSStreamingID)
	}
	log = log.WithField("message_id", msgID)
	log.WithField("tts_streaming", tmp).Debugf("Initialized the tts streaming say. tts_streaming_id: %s", cc.TTSStreamingID)

	var wg sync.WaitGroup
	errs := make(chan error, 2)

	// run response text handler
	wg.Add(1)
	go func() {
		defer wg.Done()
		tmp, err := h.streamingSendResponseHandleText(ctx, cc, msgID, chanText)
		if err != nil {
			errs <- errors.Wrapf(err, "could not handle the text response")
			return
		}
		log.WithField("response_message", tmp).Debugf("Handled the text response message. message: %s", tmp.Content)
	}()

	// run response tool handler
	wg.Add(1)
	go func() {
		defer wg.Done()
		tmp, err := h.streamingSendResponseHandleTool(ctx, cc, chanTool)
		if err != nil {
			errs <- errors.Wrapf(err, "could not handle the tool response")
			return
		}
		log.WithField("response_message", tmp).Debugf("Handled the text response tool. message: %s", tmp.Content)
	}()

	// wait for all handlers to finish
	wg.Wait()
	close(errs)
	errFlag := false
	for err := range errs {
		log.WithField("error", err).Errorf("Could not handle the response. err: %v", err)
		errFlag = true
	}
	if errFlag {
		return nil, fmt.Errorf("error occurred during response handling")
	}

	tmpFinish, err := h.reqHandler.TTSV1StreamingSayFinish(ctx, cc.TTSStreamingPodID, cc.TTSStreamingID, msgID)
	if err != nil {
		log.Errorf("Could not finish the tts streaming say. err: %v", err)
		return nil, errors.Wrapf(err, "could not finish the tts streaming say. tts_streaming_id: %s", cc.TTSStreamingID)
	}
	log.WithField("tts_streaming", tmpFinish).Debugf("Finished the tts streaming say. tts_streaming_id: %s", cc.TTSStreamingID)

	return res, nil
}

func (h *messageHandler) streamingSendOpenai(ctx context.Context, cc *aicall.AIcall) (<-chan string, <-chan *fmaction.Action, error) {

	switch cc.ReferenceType {
	case aicall.ReferenceTypeCall:
		return h.streamingSendOpenaiReferenceTypeCall(ctx, cc)

	default:
		return nil, nil, fmt.Errorf("unsupported reference type: %s", cc.ReferenceType)
	}
}

func (h *messageHandler) streamingSendOpenaiReferenceTypeCall(ctx context.Context, cc *aicall.AIcall) (<-chan string, <-chan *fmaction.Action, error) {
	filters := map[string]string{
		"deleted": "false",
	}

	// note: because of chatgpt needs entire message history, we need to send all messages
	messages, err := h.Gets(ctx, cc.ID, 1000, "", filters)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not get the messages correctly")
	}

	slices.Reverse(messages)
	chanMsg, chanAction, err := h.engineOpenaiHandler.StreamingSend(ctx, cc, messages)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not send the message correctly")
	}

	return chanMsg, chanAction, nil
}

func (h *messageHandler) streamingSendResponseHandleText(ctx context.Context, cc *aicall.AIcall, msgID uuid.UUID, chanText <-chan string) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "streamingSendResponseHandleText",
		"aicall_id": cc.ID,
	})

	totalMessage := ""
	for msg := range chanText {
		log.Debugf("Sending the streaming message to tts streaming. message: %s", msg)
		if errAdd := h.reqHandler.TTSV1StreamingSayAdd(ctx, cc.TTSStreamingPodID, cc.TTSStreamingID, msgID, msg); errAdd != nil {
			return nil, errors.Wrapf(errAdd, "could not add the text via tts streaming. tts_streaming_id: %s", cc.TTSStreamingID)
		}

		totalMessage += msg
	}
	log.Debugf("Finished sending the streaming message to tts streaming. total_message: %s", totalMessage)

	// create a message for incoming(response)
	res, err := h.Create(ctx, msgID, cc.CustomerID, cc.ID, message.DirectionIncoming, message.RoleAssistant, totalMessage)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the received message correctly")
	}
	log.WithField("response", res).Debugf("Created the response message. message_id: %s", res.ID)

	return res, nil
}

func (h *messageHandler) streamingSendResponseHandleTool(ctx context.Context, cc *aicall.AIcall, chanAction <-chan *fmaction.Action) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "streamingSendResponseHandleTool",
		"aicall_id": cc.ID,
	})

	actions := []fmaction.Action{}
	for act := range chanAction {
		log.WithField("action", act).Debugf("Received action from the ai. aicall_id: %s", cc.ID)
		actions = append(actions, *act)
	}
	if len(actions) == 0 {
		// nothing todo
		return nil, nil
	}
	log.WithField("actions", actions).Debugf("Received actions from the ai. aicall_id: %s", cc.ID)

	af, err := h.reqHandler.FlowV1ActiveflowAddActions(ctx, cc.ActiveflowID, actions)
	if err != nil {
		return nil, errors.Wrapf(err, "could not add actions to the activeflow. activeflow_id: %s", cc.ActiveflowID)
	}
	log.WithField("activeflow", af).Debugf("Added actions to the activeflow. activeflow_id: %s", cc.ActiveflowID)

	tmpContent, err := json.Marshal(actions)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal the actions")
	}

	msgID := h.utilHandler.UUIDCreate()
	res, errCreate := h.Create(ctx, msgID, cc.CustomerID, cc.ID, message.DirectionNone, message.RoleTool, string(tmpContent))
	if errCreate != nil {
		return nil, errors.Wrapf(errCreate, "could not create the tool message")
	}
	log.WithField("message", res).Debugf("Created the tool message for the actions. message_id: %s", res.ID)

	// send the terminate signal to aicall
	tmp, err := h.reqHandler.AIV1AIcallTerminate(ctx, cc.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not terminate the aicall. aicall_id: %s", cc.ID)
	}
	log.WithField("aicall", tmp).Debugf("Terminating the aicall after sending the tool actions. aicall_id: %s", cc.ID)

	return res, nil
}
