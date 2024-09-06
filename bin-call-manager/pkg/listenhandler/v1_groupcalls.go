package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/pkg/listenhandler/models/request"
)

// processV1GroupcallsGet handles POST /v1/groupcalls request
// It gets list of groupcalls.
func (h *listenHandler) processV1GroupcallsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1GroupcallsGet",
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

	// get filters
	filters := h.utilHandler.URLParseFilters(u)

	tmp, err := h.groupcallHandler.Gets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get recordings. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1GroupcallsPost handles POST /v1/groupcalls request
// It creates a new groupcall.
func (h *listenHandler) processV1GroupcallsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 2 {
		return simpleResponse(400), nil
	}

	var req request.V1DataGroupcallsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	// start groupcall
	tmp, err := h.groupcallHandler.Start(
		ctx,
		req.ID,
		req.CustomerID,
		req.FlowID,
		&req.Source,
		req.Destinations,
		req.MasterCallID,
		req.MasterGroupcallID,
		req.RingMethod,
		req.AnswerMethod,
	)
	if err != nil {
		log.Debugf("Could not create a outgoing call. err: %v", err)
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

// processV1GroupcallsIDGet handles GET /v1/groupcalls/<groupcall-id> request
// It returns a groupcall.
func (h *listenHandler) processV1GroupcallsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1GroupcallsIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.groupcallHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get groupcall info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1GroupcallsIDDelete handles DELETE /v1/groupcalls/<groupcall-id> request
// It deletes the groupcall.
func (h *listenHandler) processV1GroupcallsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1GroupcallsIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.groupcallHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not get groupcall info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1GroupcallsIDHangupPost handles POST /v1/groupcalls/<groupcall-id>/hangup request
// It hangup the groupcall.
func (h *listenHandler) processV1GroupcallsIDHangupPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1GroupcallsIDHangupPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.groupcallHandler.Hangingup(ctx, id)
	if err != nil {
		log.Errorf("Could not get groupcall info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1GroupcallsIDAnswerGroupcallIDPost handles POST /v1/groupcalls/<groupcall-id>/answer_groupcall_id request
// It hangup the groupcall.
func (h *listenHandler) processV1GroupcallsIDAnswerGroupcallIDPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1GroupcallsIDAnswerGroupcallIDPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	var req request.V1DataGroupcallsIDAnswerGroupcallIDPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.groupcallHandler.AnswerGroupcall(ctx, id, req.AnswerGroupcallID)
	if err != nil {
		log.Errorf("Could not get groupcall info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1GroupcallsIDHangupGroupcallPost handles POST /v1/groupcalls/<groupcall-id>/hangup_groupcall request
// It handles hangup the groupcall.
func (h *listenHandler) processV1GroupcallsIDHangupGroupcallPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1GroupcallsIDHangupGroupcallPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.groupcallHandler.HangupGroupcall(ctx, id)
	if err != nil {
		log.Errorf("Could not get groupcall info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1GroupcallsIDHangupCallPost handles POST /v1/groupcalls/<groupcall-id>/hangup_call request
// It handles hangup the groupcall.
func (h *listenHandler) processV1GroupcallsIDHangupCallPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1GroupcallsIDHangupCallPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.groupcallHandler.HangupCall(ctx, id)
	if err != nil {
		log.Errorf("Could not get groupcall info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
