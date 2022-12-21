package listenhandler

// // processV1CallRecordingsPost handles POST /v1/call-recordings request
// // It creates a new speech-to-text.
// func (h *listenHandler) processV1CallRecordingsPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
// 	ctx := context.Background()
// 	uriItems := strings.Split(m.URI, "/")
// 	if len(uriItems) < 3 {
// 		return simpleResponse(400), nil
// 	}

// 	var reqData request.V1DataCallRecordingsPost
// 	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
// 		// same call-id is already exsit
// 		logrus.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
// 		return simpleResponse(400), nil
// 	}

// 	// do transcribe
// 	tmp, err := h.transcribeHandler.CallRecording(ctx, reqData.CustomerID, reqData.ReferenceID, reqData.Language)
// 	if err != nil {
// 		logrus.Debugf("Could not create transcribe. err: %v", err)
// 		return simpleResponse(500), nil
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
