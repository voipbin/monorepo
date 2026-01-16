package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/pkg/listenhandler/models/request"
)

// processV1AccountsGet handles
// /v1/accounts GET
func (h *listenHandler) processV1AccountsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsGet",
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

	var req map[string]any
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	filters, err := account.ConvertStringMapToFieldMap(req)
	if err != nil {
		log.Errorf("Could not convert the filters. err: %v", err)
		return simpleResponse(400), nil
	}

	tmps, err := h.accountHandler.List(ctx, pageToken, pageSize, filters)
	if err != nil {
		log.Debugf("Could not get conversations. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmps)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmps, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AccountsPost handles
// POST /v1/accounts request
func (h *listenHandler) processV1AccountsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsPost",
		"request": m,
	})

	var req request.V1DataAccountsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.accountHandler.Create(ctx, req.CustomerID, req.Type, req.Name, req.Detail, req.Secret, req.Token)
	if err != nil {
		log.Errorf("Could not create the account. err: %v", err)
		return nil, errors.Wrap(err, "could not create the account")
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

// processV1AccountsIDGet handles
// /v1/accounts/{id} GET
func (h *listenHandler) processV1AccountsIDGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(req.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":       "processV1AccountsIDGet",
		"account_id": id,
	})
	log.Debugf("Executing processV1AccountsIDGet. account_id: %s", id)

	tmp, err := h.accountHandler.Get(ctx, id)
	if err != nil {
		log.Debugf("Could not get a conversation. conversation_id: %s, err: %v", id, err)
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

// processV1AccountsIDPut handles
// /v1/accounts/{id} PUT
func (h *listenHandler) processV1AccountsIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsIDPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log.Debugf("Executing processV1AccountsIDPut. account_id: %s", id)

	allowedItems := []string{
		string(account.FieldName),
		string(account.FieldDetail),
		string(account.FieldType),
		string(account.FieldSecret),
		string(account.FieldToken),
	}
	filteredItems, err := requesthandler.GetFilteredItems(m, allowedItems)
	if err != nil {
		log.Errorf("Could not filter the request. err: %v", err)
		return nil, err
	}
	if len(filteredItems) == 0 {
		log.Debugf("No allowed fields provided for update. Skipping.")
		return simpleResponse(200), nil
	}

	tmpFields, err := account.ConvertStringMapToFieldMap(filteredItems)
	if err != nil {
		log.Errorf("Could not convert field map. err: %v", err)
		return nil, err
	}

	tmp, err := h.accountHandler.Update(ctx, id, tmpFields)
	if err != nil {
		log.Debugf("Could not get a conversation. conversation_id: %s, err: %v", id, err)
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

// processV1AccountsIDDelete handles
// /v1/accounts/{id} DELETE
func (h *listenHandler) processV1AccountsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log.Debugf("Executing processV1AccountsIDDelete. account_id: %s", id)

	tmp, err := h.accountHandler.Delete(ctx, id)
	if err != nil {
		log.Debugf("Could not delete the account. account_id: %s, err: %v", id, err)
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
