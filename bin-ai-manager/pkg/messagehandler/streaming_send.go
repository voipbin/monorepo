package messagehandler

// func (h *messageHandler) StreamingSend(ctx context.Context, aicallID uuid.UUID, role message.Role, content string) (*message.Message, error) {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func":      "StreamingSend",
// 		"aicall_id": aicallID,
// 		"role":      role,
// 		"content":   content,
// 	})
// 	log.Debugf("Sending ai message.")

// 	cc, err := h.reqHandler.AIV1AIcallGet(ctx, aicallID)
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "could not get the aicall correctly")
// 	}

// 	if cc.Status == aicall.StatusTerminated {
// 		return nil, errors.New("aicall is already ended")
// 	} else if cc.ReferenceType != aicall.ReferenceTypeCall {
// 		return nil, fmt.Errorf("unsupported reference type: %s", cc.ReferenceType)
// 	}

// 	// create a message for outgoing(request)
// 	res, err := h.Create(ctx, uuid.Nil, cc.CustomerID, aicallID, message.DirectionOutgoing, role, content, nil, "")
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "could not create the sending message correctly")
// 	}

// 	// send
// 	if errSend := h.streamingSend(ctx, cc); errSend != nil {
// 		return nil, errors.Wrapf(errSend, "could not send the message correctly")
// 	}

// 	return res, nil
// }

// // StreamingSendAll sends all messages of the given aicall to the ai engine
// // used for tool response back to the ai engine
// func (h *messageHandler) StreamingSendAll(ctx context.Context, aicallID uuid.UUID) error {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func":      "StreamingSendAll",
// 		"aicall_id": aicallID,
// 	})
// 	log.Debugf("Sending all ai messages.")

// 	cc, err := h.reqHandler.AIV1AIcallGet(ctx, aicallID)
// 	if err != nil {
// 		return errors.Wrapf(err, "could not get the aicall correctly")
// 	}

// 	if cc.Status == aicall.StatusTerminated {
// 		return errors.New("aicall is already ended")
// 	} else if cc.ReferenceType != aicall.ReferenceTypeCall {
// 		return fmt.Errorf("unsupported reference type: %s", cc.ReferenceType)
// 	}

// 	go func() {
// 		// note: we're running this in a goroutine to not block the caller
// 		if errSend := h.streamingSend(ctx, cc); errSend != nil {
// 			log.Errorf("Could not send all the messages after tool action. err: %v", errSend)
// 		}
// 	}()

// 	return nil
// }

// // streamingSend sends the given aicall's messages to the ai engine and handles the response
// func (h *messageHandler) streamingSend(ctx context.Context, cc *aicall.AIcall) error {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func":    "streamingSend",
// 		"ai_call": cc,
// 	})
// 	log.Debugf("Sending ai message.")

// 	t1 := time.Now()
// 	var chanText <-chan string
// 	var chanTool <-chan *message.ToolCall
// 	var err error

// 	modelTarget := ai.GetEngineModelTarget(cc.AIEngineModel)
// 	switch modelTarget {
// 	case ai.EngineModelTargetOpenAI:
// 		chanText, chanTool, err = h.streamingSendOpenai(ctx, cc)

// 	default:
// 		err = fmt.Errorf("unsupported ai engine model: %s", cc.AIEngineModel)
// 	}
// 	if err != nil {
// 		return errors.Wrapf(err, "could not send the message correctly")
// 	}

// 	t2 := time.Since(t1)
// 	promMessageProcessTime.WithLabelValues(string(cc.AIEngineType)).Observe(float64(t2.Milliseconds()))

// 	msgID := h.utilHandler.UUIDCreate()
// 	tmp, err := h.reqHandler.TTSV1StreamingSayInit(ctx, cc.TTSStreamingPodID, cc.TTSStreamingID, msgID)
// 	if err != nil {
// 		return errors.Wrapf(err, "could not say the text via tts streaming. tts_streaming_id: %s", cc.TTSStreamingID)
// 	}
// 	log = log.WithField("message_id", msgID)
// 	log.WithField("tts_streaming", tmp).Debugf("Initialized the tts streaming say. tts_streaming_id: %s", cc.TTSStreamingID)

// 	var wg sync.WaitGroup
// 	errs := make(chan error, 2)

// 	// run response text handler
// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		tmp, err := h.streamingSendResponseHandleText(ctx, cc, msgID, chanText)
// 		if err != nil {
// 			errs <- errors.Wrapf(err, "could not handle the text response")
// 			return
// 		}

// 		if tmp == nil {
// 			return
// 		}
// 		log.WithField("response_message", tmp).Debugf("Handled the text response message. message: %s", tmp.Content)
// 	}()

// 	// run response tool handler
// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		tmp, err := h.streamingSendResponseHandleTool(ctx, cc, chanTool)
// 		if err != nil {
// 			errs <- errors.Wrapf(err, "could not handle the tool response")
// 			return
// 		}

// 		if tmp == nil {
// 			return
// 		}
// 		log.WithField("response_message", tmp).Debugf("Handled the text response tool. message: %s", tmp.Content)
// 	}()

// 	// wait for all handlers to finish
// 	wg.Wait()
// 	close(errs)
// 	errFlag := false
// 	for err := range errs {
// 		log.WithField("error", err).Errorf("Could not handle the response. err: %v", err)
// 		errFlag = true
// 	}
// 	if errFlag {
// 		return fmt.Errorf("error occurred during response handling")
// 	}

// 	tmpFinish, err := h.reqHandler.TTSV1StreamingSayFinish(ctx, cc.TTSStreamingPodID, cc.TTSStreamingID, msgID)
// 	if err != nil {
// 		log.Errorf("Could not finish the tts streaming say. err: %v", err)
// 		return errors.Wrapf(err, "could not finish the tts streaming say. tts_streaming_id: %s", cc.TTSStreamingID)
// 	}
// 	log.WithField("tts_streaming", tmpFinish).Debugf("Finished the tts streaming say. tts_streaming_id: %s", cc.TTSStreamingID)

// 	return nil
// }

// func (h *messageHandler) streamingSendOpenai(ctx context.Context, cc *aicall.AIcall) (<-chan string, <-chan *message.ToolCall, error) {

// 	switch cc.ReferenceType {
// 	case aicall.ReferenceTypeCall:
// 		return h.streamingSendOpenaiReferenceTypeCall(ctx, cc)

// 	default:
// 		return nil, nil, fmt.Errorf("unsupported reference type: %s", cc.ReferenceType)
// 	}
// }

// func (h *messageHandler) streamingSendOpenaiReferenceTypeCall(ctx context.Context, cc *aicall.AIcall) (<-chan string, <-chan *message.ToolCall, error) {
// 	filters := map[string]string{
// 		"deleted": "false",
// 	}

// 	// note: because of chatgpt needs entire message history, we need to send all messages
// 	messages, err := h.Gets(ctx, cc.ID, 1000, "", filters)
// 	if err != nil {
// 		return nil, nil, errors.Wrapf(err, "could not get the messages correctly")
// 	}

// 	slices.Reverse(messages)
// 	chanMsg, chanAction, err := h.engineOpenaiHandler.StreamingSend(ctx, cc, messages)
// 	if err != nil {
// 		return nil, nil, errors.Wrapf(err, "could not send the message correctly")
// 	}

// 	return chanMsg, chanAction, nil
// }

// func (h *messageHandler) streamingSendResponseHandleText(ctx context.Context, cc *aicall.AIcall, msgID uuid.UUID, chanText <-chan string) (*message.Message, error) {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func":      "streamingSendResponseHandleText",
// 		"aicall_id": cc.ID,
// 	})

// 	totalMessage := ""
// 	for msg := range chanText {
// 		log.Debugf("Sending the streaming message to tts streaming. message: %s", msg)
// 		if errAdd := h.reqHandler.TTSV1StreamingSayAdd(ctx, cc.TTSStreamingPodID, cc.TTSStreamingID, msgID, msg); errAdd != nil {
// 			return nil, errors.Wrapf(errAdd, "could not add the text via tts streaming. tts_streaming_id: %s", cc.TTSStreamingID)
// 		}

// 		totalMessage += msg
// 	}
// 	log.Debugf("Finished sending the streaming message to tts streaming. total_message: %s", totalMessage)

// 	if totalMessage == "" {
// 		// nothing to do
// 		return nil, nil
// 	}

// 	// create a message for incoming(response)
// 	res, err := h.Create(ctx, msgID, cc.CustomerID, cc.ID, message.DirectionIncoming, message.RoleAssistant, totalMessage, nil, "")
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "could not create the received message correctly")
// 	}
// 	log.WithField("response", res).Debugf("Created the response message. message_id: %s", res.ID)

// 	return res, nil
// }

// func (h *messageHandler) streamingSendResponseHandleTool(ctx context.Context, cc *aicall.AIcall, chanToolCall <-chan *message.ToolCall) (*message.Message, error) {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func":      "streamingSendResponseHandleTool",
// 		"aicall_id": cc.ID,
// 	})

// 	// gather all tool calls first
// 	toolCalls := []message.ToolCall{}
// 	for toolCall := range chanToolCall {
// 		toolCalls = append(toolCalls, *toolCall)
// 	}

// 	if len(toolCalls) == 0 {
// 		// nothing todo
// 		return nil, nil
// 	}

// 	// create a message for tool call request
// 	res, errCreate := h.Create(ctx, uuid.Nil, cc.CustomerID, cc.ID, message.DirectionIncoming, message.RoleAssistant, "", toolCalls, "")
// 	if errCreate != nil {
// 		return nil, errors.Wrapf(errCreate, "could not create the tool message")
// 	}
// 	log.WithField("message", res).Debugf("Created the tool message for the actions. message_id: %s", res.ID)

// 	terminate := false
// 	for _, toolCall := range toolCalls {
// 		tmpTerminate, err := h.toolMessageHandle(ctx, cc, &toolCall)
// 		if err != nil {
// 			log.WithField("tool_call", toolCall).Errorf("Could not handle the tool call correctly. err: %v", err)
// 			continue
// 		}

// 		if tmpTerminate {
// 			terminate = true
// 		}
// 	}

// 	if terminate {
// 		// we've got a terminate signal from the tool action, so we need to terminate the aicall
// 		// this will stop the aicall and will continue the next action in the activeflow
// 		// send the terminate signal to aicall
// 		tmp, err := h.reqHandler.AIV1AIcallTerminate(ctx, cc.ID)
// 		if err != nil {
// 			return nil, errors.Wrapf(err, "could not terminate the aicall. aicall_id: %s", cc.ID)
// 		}
// 		log.WithField("aicall", tmp).Debugf("Terminating the aicall after sending the tool actions. aicall_id: %s", cc.ID)
// 		return res, nil
// 	}

// 	// note: we've just processed tool actions, so we need to send all the messages again to the ai engine
// 	if errSend := h.reqHandler.AIV1AIcallSendAll(ctx, cc.ID); errSend != nil {
// 		// we're logging the error here, but we're not returning it
// 		log.Errorf("Could not send all the messages after tool action. err: %v", errSend)
// 	}

// 	return res, nil
// }
