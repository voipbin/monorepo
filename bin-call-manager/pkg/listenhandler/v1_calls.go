package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/pkg/listenhandler/models/request"
	"monorepo/bin-call-manager/pkg/listenhandler/models/response"
)

// processV1CallsGet handles GET /v1/calls request
func (h *listenHandler) processV1CallsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get filters from request body
	tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
	if err != nil {
		log.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	// convert to typed filters
	filters, err := utilhandler.ConvertFilters[call.FieldStruct, call.Field](call.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	calls, err := h.callHandler.Gets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get calls. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(calls)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", calls, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDGet handles GET /v1/calls/<call-id> request
func (h *listenHandler) processV1CallsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	c, err := h.callHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(c)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsPost handles POST /v1/calls request
// It creates a new call.
func (h *listenHandler) processV1CallsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 3 {
		return simpleResponse(400), nil
	}

	var req request.V1DataCallsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	calls, groupcalls, err := h.callHandler.CreateCallsOutgoing(ctx, req.CustomerID, req.FlowID, req.MasterCallID, req.Source, req.Destinations, req.EarlyExecution, req.Connect)
	if err != nil {
		log.Debugf("Could not create a outgoing call. err: %v", err)
		return simpleResponse(500), nil
	}

	tmp := &response.V1ResponseCallsPost{
		Calls:      calls,
		Groupcalls: groupcalls,
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDPost handles POST /v1/calls/<call-id> request
// It creates a new call.
func (h *listenHandler) processV1CallsIDPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataCallsIDPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	c, err := h.callHandler.CreateCallOutgoing(ctx, id, req.CustomerID, req.FlowID, req.ActiveflosID, req.MasterCallID, req.GroupcallID, req.Source, req.Destination, req.EarlyExecution, req.Connect)
	if err != nil {
		log.Debugf("Could not create a outgoing call. flow: %s, source: %v, destination: %v, err: %v", req.FlowID, req.Source, req.Destination, err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(c)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", c, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDDelete handles Post /v1/calls/<call-id> request
// It hangs up the call.
func (h *listenHandler) processV1CallsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	// delete the call
	tmp, err := h.callHandler.Delete(ctx, id)
	if err != nil {
		log.Debugf("Could not delete the call. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDHangupPost handles Post /v1/calls/<call-id>/hangup request
// It hangs up the call.
func (h *listenHandler) processV1CallsIDHangupPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDHangupPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	// hanging up the call
	tmp, err := h.callHandler.HangingUp(ctx, id, call.HangupReasonNormal)
	if err != nil {
		log.Debugf("Could not hanging up the call. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDHealthPost handles /v1/calls/<call-id>/health-check request
func (h *listenHandler) processV1CallsIDHealthPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDHealthPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataCallsIDHealthPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not marshal the request message. message: %v, err: %v", req, err)
		return nil, err
	}

	go h.callHandler.HealthCheck(ctx, id, req.RetryCount)
	return nil, nil
}

// processV1CallsIDGet handles /v1/calls/<call-id>/action-timeout request
func (h *listenHandler) processV1CallsIDActionTimeoutPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDActionTimeoutPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")

	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var data request.V1DataCallsIDActionTimeoutPost
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		return nil, err
	}

	action := &fmaction.Action{
		ID:        data.ActionID,
		Type:      data.ActionType,
		TMExecute: data.TMExecute,
	}

	if err := h.callHandler.ActionTimeout(ctx, id, action); err != nil {
		log.Debugf("Could not handle the action timeout request. err: %v", err)
		return simpleResponse(404), nil
	}

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDGet handles /v1/calls/<call-id>/action-next request
func (h *listenHandler) processV1CallsIDActionNextPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDActionNextPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var data request.V1DataCallsIDActionNextPost
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		return nil, err
	}

	c, err := h.callHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info from the database. err: %v", err)
		return simpleResponse(404), nil
	}

	// we run the go runc() here.
	// because we don't want to action's running time caused the request timeout.
	go func() {
		if data.Force {
			if err := h.callHandler.ActionNextForce(ctx, c); err != nil {
				log.Errorf("Could not execute the action next force. err: %v", err)
			}
		} else {
			if err := h.callHandler.ActionNext(ctx, c); err != nil {
				log.Errorf("Could not execute the action next. err: %v", err)
			}
		}
	}()

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDChainedCallIDsPost handles /v1/calls/<call-id>/chained-call-ids POST request
func (h *listenHandler) processV1CallsIDChainedCallIDsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDChainedCallIDsPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataCallsIDChainedCallIDsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	tmp, err := h.callHandler.ChainedCallIDAdd(ctx, id, req.ChainedCallID)
	if err != nil {
		log.Debugf("Could not get updated call info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDChainedCallIDsDelete handles /v1/calls/<call-id>/chained-call-ids/<chained-call-id> DELETE request
func (h *listenHandler) processV1CallsIDChainedCallIDsDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDChainedCallIDsDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	chainedCallID := uuid.FromStringOrNil(uriItems[5])

	tmp, err := h.callHandler.ChainedCallIDRemove(ctx, id, chainedCallID)
	if err != nil {
		log.Debugf("Could not get updated call info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDExternalMediaPost handles /v1/calls/<call-id>/external-media POST request
func (h *listenHandler) processV1CallsIDExternalMediaPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDExternalMediaPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataCallsIDExternalMediaPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	tmp, err := h.callHandler.ExternalMediaStart(
		ctx,
		id,
		req.ExternalMediaID,
		req.ExternalHost,
		externalmedia.Encapsulation(req.Encapsulation),
		externalmedia.Transport(req.Transport),
		req.ConnectionType,
		req.Format,
		req.DirectionListen,
		req.DirectionSpeak,
	)
	if err != nil {
		log.Errorf("Could not start the external media. call: %s, err: %v", id, err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDExternalMediaDelete handles /v1/calls/<call-id>/external-media DELETE request
func (h *listenHandler) processV1CallsIDExternalMediaDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDExternalMediaDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.callHandler.ExternalMediaStop(ctx, id)
	if err != nil {
		log.Errorf("Could not stop the external media. call_id: %s, err: %v", id, err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDDigitsGet handles /v1/calls/<call-id>/digits GET request
func (h *listenHandler) processV1CallsIDDigitsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDDigitsGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	digit, err := h.callHandler.DigitsGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get call's digits. err: %v", err)
		return simpleResponse(500), nil
	}

	tmp := &response.V1ResponseCallsIDDigitsGet{
		Digits: digit,
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDDigitsSet handles /v1/calls/<call-id>/digits POST request
func (h *listenHandler) processV1CallsIDDigitsSet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDDigitsSet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataCallsIDDigitsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	if err := h.callHandler.DigitsSet(ctx, id, req.Digits); err != nil {
		log.Errorf("Could not get call's digits. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1CallsIDRecordingIDPut handles /v1/calls/<call-id>/recording_id PUT request
func (h *listenHandler) processV1CallsIDRecordingIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDRecordingIDPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataCallsIDRecordingIDPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	tmp, err := h.callHandler.UpdateRecordingID(ctx, id, req.RecordingID)
	if err != nil {
		log.Errorf("Could not update call's recording id. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDConfbridgeIDPut handles /v1/calls/<call-id>/confbridge_id PUT request
func (h *listenHandler) processV1CallsIDConfbridgeIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDConfbridgeIDPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataCallsIDConfbridgeIDPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	tmp, err := h.callHandler.UpdateConfbridgeID(ctx, id, req.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not update call's recording id. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDRecordingStartPost handles /v1/calls/<call-id>/recording_start POST request
func (h *listenHandler) processV1CallsIDRecordingStartPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDRecordingStartPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataCallsIDRecordingStartPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	tmp, err := h.callHandler.RecordingStart(ctx, id, req.Format, req.EndOfSilence, req.EndOfKey, req.Duration, req.OnEndFlowID)
	if err != nil {
		log.Errorf("Could not start call recording. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDRecordingStopPost handles /v1/calls/<call-id>/recording_stop POST request
func (h *listenHandler) processV1CallsIDRecordingStopPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDRecordingStopPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.callHandler.RecordingStop(ctx, id)
	if err != nil {
		log.Errorf("Could not start call recording. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDTalkPost handles /v1/calls/<call-id>/talk POST request
func (h *listenHandler) processV1CallsIDTalkPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDTalkPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataCallsIDTalkPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	if errTalk := h.callHandler.Talk(ctx, id, false, req.Text, req.Gender, req.Language); errTalk != nil {
		log.Errorf("Could not talk to the call. err: %v", errTalk)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDPlayPost handles /v1/calls/<call-id>/play POST request
func (h *listenHandler) processV1CallsIDPlayPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDPlayPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataCallsIDPlayPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	if errPlay := h.callHandler.Play(ctx, id, false, req.MediaURLs); errPlay != nil {
		log.Errorf("Could not play the medias. err: %v", errPlay)
		return nil, errors.Wrap(errPlay, "could not play the medias")
	}

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDMediaStopPost handles /v1/calls/<call-id>/media_stop POST request
func (h *listenHandler) processV1CallsIDMediaStopPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDMediaStopPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	if errStop := h.callHandler.MediaStop(ctx, id); errStop != nil {
		log.Errorf("Could not stop the media. err: %v", errStop)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDHoldPost handles /v1/calls/<call-id>/hold POST request
func (h *listenHandler) processV1CallsIDHoldPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDHoldPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	if errHold := h.callHandler.HoldOn(ctx, id); errHold != nil {
		log.Errorf("Could not hold the call. err: %v", errHold)
		return nil, errors.Wrap(errHold, "could not hold the call")
	}

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDHoldDelete handles /v1/calls/<call-id>/hold DELETE request
func (h *listenHandler) processV1CallsIDHoldDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDHoldDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	if errHold := h.callHandler.HoldOff(ctx, id); errHold != nil {
		log.Errorf("Could not unhold the call. err: %v", errHold)
		return nil, errors.Wrap(errHold, "could not unhold the call")
	}

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDMutePost handles /v1/calls/<call-id>/mute POST request
func (h *listenHandler) processV1CallsIDMutePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDMutePost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataCallsIDMutePost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	if errMute := h.callHandler.MuteOn(ctx, id, req.Direction); errMute != nil {
		log.Errorf("Could not hold the call. err: %v", errMute)
		return nil, errors.Wrap(errMute, "could not hold the call")
	}

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDMuteDelete handles /v1/calls/<call-id>/mute DELETE request
func (h *listenHandler) processV1CallsIDMuteDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDMuteDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataCallsIDMuteDelete
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	if errMute := h.callHandler.MuteOff(ctx, id, req.Direction); errMute != nil {
		log.Errorf("Could not unmute the call. err: %v", errMute)
		return nil, errors.Wrap(errMute, "could not unmute the call")
	}

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDMOHPost handles /v1/calls/<call-id>/moh POST request
func (h *listenHandler) processV1CallsIDMOHPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDMOHPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	if errMOH := h.callHandler.MOHOn(ctx, id); errMOH != nil {
		log.Errorf("Could not hold the call. err: %v", errMOH)
		return nil, errors.Wrap(errMOH, "could not hold the call")
	}

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDMOHDelete handles /v1/calls/<call-id>/moh DELETE request
func (h *listenHandler) processV1CallsIDMOHDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDMOHDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	if errMOH := h.callHandler.MOHOff(ctx, id); errMOH != nil {
		log.Errorf("Could not moh off the call. err: %v", errMOH)
		return nil, errors.Wrap(errMOH, "could not moh off the call")
	}

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDSilencePost handles /v1/calls/<call-id>/silence POST request
func (h *listenHandler) processV1CallsIDSilencePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDSilencePost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	if errSilence := h.callHandler.SilenceOn(ctx, id); errSilence != nil {
		log.Errorf("Could not silence the call. err: %v", errSilence)
		return nil, errors.Wrap(errSilence, "could not silence the call")
	}

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDSilenceDelete handles /v1/calls/<call-id>/silence DELETE request
func (h *listenHandler) processV1CallsIDSilenceDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsIDSilenceDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	if errSilence := h.callHandler.SilenceOff(ctx, id); errSilence != nil {
		log.Errorf("Could not silence off the call. err: %v", errSilence)
		return nil, errors.Wrap(errSilence, "could not silence off the call")
	}

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}
