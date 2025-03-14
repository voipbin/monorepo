package listenhandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-email-manager/pkg/listenhandler/models/request"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// v1EmailsGet handles /v1/emails GET request
func (h *listenHandler) v1EmailsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// parse the filters
	filters := h.utilHandler.URLParseFilters(u)

	tmp, err := h.emailHandler.Gets(ctx, pageToken, pageSize, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get emails")
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal the res")
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1EmailsPost handles /v1/emails POST request
// creates a new email with given data.
func (h *listenHandler) v1EmailsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	var req request.V1DataEmailsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		return nil, errors.Errorf("could not marshal the data")
	}

	tmp, err := h.emailHandler.Create(ctx, req.CustomerID, req.ActiveflowID, req.Destinations, req.Subject, req.Content, req.Attachments)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create email")
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal the res")
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1EmailsIDGet handles
// /v1/emails/<email-id> GET
func (h *listenHandler) v1EmailsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	// "/v1/emails/be2692f8-066a-11eb-847f-1b4de696fafb"
	tmpVals := strings.Split(m.URI, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.emailHandler.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get email")
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal the res")
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ActiveflowsIDDelete handles
// /v1/emails/{id} DELETE
func (h *listenHandler) v1EmailsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	// "/v1/emails/be2692f8-066a-11eb-847f-1b4de696fafb"
	tmpVals := strings.Split(m.URI, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.emailHandler.Delete(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not delete email")
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal the res")
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
