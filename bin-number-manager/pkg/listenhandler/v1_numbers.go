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

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/pkg/listenhandler/models/request"
)

// processV1NumbersPost handles POST /v1/numbers request
func (h *listenHandler) processV1NumbersPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1NumbersPost",
		"request": m,
	})

	var req request.V1DataNumbersPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		logrus.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	numb, err := h.numberHandler.Create(ctx, req.CustomerID, req.Number, req.CallFlowID, req.MessageFlowID, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not handle the order number. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(numb)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", numb, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1NumbersIDDelete handles DELETE /v1/numbers/<id> request
func (h *listenHandler) processV1NumbersIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1NumbersIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log.Debugf("Executing processV1OrderNumbersIDDelete. number: %s", id)

	number, err := h.numberHandler.Delete(ctx, id)
	if err != nil {
		log.Debugf("Could not delete the number. number: %s, err: %v", id, err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(number)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", number, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1NumbersIDGet handles GET /v1/numbers/<id> request
func (h *listenHandler) processV1NumbersIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1NumbersIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log.Debugf("Executing processV1OrderNumbersIDGet. number: %s", id)

	number, err := h.numberHandler.Get(ctx, id)
	if err != nil {
		log.Debugf("Could not get a number. number: %s, err: %v", id, err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(number)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", number, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1NumbersIDPut handles PUT /v1/numbers/<id> request
func (h *listenHandler) processV1NumbersIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1NumbersIDPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataNumbersIDPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	log.Debugf("Executing processV1NumbersIDPut. number: %s", id)

	fields := map[number.Field]any{
		number.FieldCallFlowID:    req.CallFlowID,
		number.FieldMessageFlowID: req.MessageFlowID,
		number.FieldName:          req.Name,
		number.FieldDetail:        req.Detail,
	}

	num, err := h.numberHandler.Update(ctx, id, fields)
	if err != nil {
		log.Debugf("Could not update the number. number: %s, err: %v", id, err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(num)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", num, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1NumbersGet handles GET /v1/numbers request
func (h *listenHandler) processV1NumbersGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1NumbersGet",
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
	// parse the filters
	stringFilters := h.utilHandler.URLParseFilters(u)

	// Convert string filters to typed filters
	filters := convertToTypedFilters(stringFilters)

	numbers, err := h.numberHandler.Gets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Debugf("Could not get numbers. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(numbers)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", numbers, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// convertToTypedFilters converts string filters to typed filters
func convertToTypedFilters(stringFilters map[string]string) map[number.Field]any {
	filters := make(map[number.Field]any)

	for k, v := range stringFilters {
		switch k {
		case "customer_id", "call_flow_id", "message_flow_id":
			filters[number.Field(k)] = uuid.FromStringOrNil(v)
		case "deleted":
			filters[number.FieldDeleted] = v == "true"
		default:
			filters[number.Field(k)] = v
		}
	}

	return filters
}

// processV1NumbersIDFlowIDsPut handles PUT /v1/numbers/<id>/flow_id request
func (h *listenHandler) processV1NumbersIDFlowIDsPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1NumbersIDFlowIDsPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataNumbersIDFlowIDPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		logrus.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	log.Debugf("Executing processV1NumbersIDFlowIDsPut. number: %s", id)

	fields := map[number.Field]any{
		number.FieldCallFlowID:    req.CallFlowID,
		number.FieldMessageFlowID: req.MessageFlowID,
	}

	num, err := h.numberHandler.Update(ctx, id, fields)
	if err != nil {
		log.Debugf("Could not update the number's flow_id. number_id: %s, err: %v", id, err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(num)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", num, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1NumbersRenewPost handles POST /v1/numbers/renew request
func (h *listenHandler) processV1NumbersRenewPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1NumbersRenewPost",
		"request": m,
	})

	var req request.V1DataNumbersRenewPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		logrus.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	numb, err := h.numberHandler.RenewNumbers(ctx, req.Days, req.Hours, req.TMRenew)
	if err != nil {
		log.Errorf("Could not handle the order number. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(numb)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", numb, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
