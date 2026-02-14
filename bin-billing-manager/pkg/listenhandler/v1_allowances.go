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
)

// processV1AccountsIDAllowancesGet handles GET /v1/accounts/<account-id>/allowances request
func (h *listenHandler) processV1AccountsIDAllowancesGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsIDAllowancesGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	accountID := uuid.FromStringOrNil(uriItems[3])

	u, err := url.Parse(m.URI)
	if err != nil {
		return simpleResponse(400), nil
	}

	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	if pageSize == 0 {
		pageSize = 10
	}
	pageToken := u.Query().Get(PageToken)

	allowances, err := h.allowanceHandler.ListByAccountID(ctx, accountID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get allowances. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(allowances)
	if err != nil {
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
