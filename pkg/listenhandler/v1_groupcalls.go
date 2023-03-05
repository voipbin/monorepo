package listenhandler

// // processV1GroupcallsPost handles POST /v1/groupcalls request
// // It creates a new groupcall.
// func (h *listenHandler) processV1GroupcallsPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func":    "processV1CallsPost",
// 		"request": m,
// 	})

// 	uriItems := strings.Split(m.URI, "/")
// 	if len(uriItems) < 3 {
// 		return simpleResponse(400), nil
// 	}

// 	var req request.V1DataCallsPost
// 	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
// 		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
// 		return simpleResponse(400), nil
// 	}

// 	calls, err := h.callHandler.CreateCallsOutgoing(ctx, req.CustomerID, req.FlowID, req.MasterCallID, req.Source, req.Destinations, req.EarlyExecution, req.ExecuteNextMasterOnHangup)
// 	if err != nil {
// 		log.Debugf("Could not create a outgoing call. err: %v", err)
// 		return simpleResponse(500), nil
// 	}

// 	data, err := json.Marshal(calls)
// 	if err != nil {
// 		log.Debugf("Could not marshal the response message. message: %v, err: %v", calls, err)
// 		return simpleResponse(500), nil
// 	}

// 	res := &rabbitmqhandler.Response{
// 		StatusCode: 200,
// 		DataType:   "application/json",
// 		Data:       data,
// 	}

// 	return res, nil
// }
