package listenhandler

// // processV1StreamingsPost handles POST /v1/streamings request
// // It creates a new speech-to-text.
// func (h *listenHandler) processV1StreamingsPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
// 	uriItems := strings.Split(m.URI, "/")
// 	if len(uriItems) < 3 {
// 		return simpleResponse(400), nil
// 	}

// 	var reqData request.V1DataStreamingsPost
// 	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
// 		logrus.Errorf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
// 		return simpleResponse(400), nil
// 	}

// 	// do transcribe
// 	ctx := context.Background()
// 	tmp, err := h.transcribeHandler.StreamingTranscribeStart(ctx, reqData.CustomerID, reqData.ReferenceID, reqData.ReferenceType, reqData.Language)
// 	if err != nil {
// 		return simpleResponse(400), nil
// 	}

// 	d, err := json.Marshal(tmp)
// 	if err != nil {
// 		logrus.Errorf("Could not marshal the data. err: %v", err)
// 		return simpleResponse(500), nil
// 	}

// 	res := &rabbitmqhandler.Response{
// 		StatusCode: 200,
// 		DataType:   "application/json",
// 		Data:       d,
// 	}

// 	return res, nil
// }

// // processV1StreamingsPost handles Delete /v1/streamings/<id> request
// // It stops speech-to-text.
// func (h *listenHandler) processV1StreamingsIDDelete(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
// 	uriItems := strings.Split(m.URI, "/")
// 	if len(uriItems) < 4 {
// 		return simpleResponse(400), nil
// 	}

// 	id := uuid.FromStringOrNil(uriItems[3])
// 	log := logrus.WithFields(
// 		logrus.Fields{
// 			"id": id,
// 		})
// 	log.Debug("Executing processV1StreamingsIDDelete.")

// 	ctx := context.Background()
// 	if err := h.transcribeHandler.StreamingTranscribeStop(ctx, id); err != nil {
// 		log.Errorf("Could not stop the transcribe. err: %v", err)
// 		return simpleResponse(400), nil
// 	}

// 	return simpleResponse(200), nil
// }
