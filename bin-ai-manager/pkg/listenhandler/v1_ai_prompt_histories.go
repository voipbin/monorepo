package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
)

// processV1AIsIDPromptHistoriesGet handles GET /v1/ais/<ai-id>/prompt_histories?...
func (h *listenHandler) processV1AIsIDPromptHistoriesGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIsIDPromptHistoriesGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		log.Errorf("Could not parse the request uri. err: %v", err)
		return simpleResponse(400), nil
	}

	uriItems := strings.Split(u.Path, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	aiID := uuid.FromStringOrNil(uriItems[3])
	if aiID == uuid.Nil {
		log.Errorf("Invalid AI ID.")
		return simpleResponse(400), nil
	}

	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	tmp, err := h.aiprompthistoryHandler.List(ctx, aiID, pageSize, pageToken)
	if err != nil {
		log.Debugf("Could not get items. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

// processV1AIsIDPromptHistoriesIDGet handles GET /v1/ais/<ai-id>/prompt_histories/<history-id>
func (h *listenHandler) processV1AIsIDPromptHistoriesIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIsIDPromptHistoriesIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 6 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	aiID := uuid.FromStringOrNil(uriItems[3])
	if aiID == uuid.Nil {
		log.Errorf("Invalid AI ID.")
		return simpleResponse(400), nil
	}
	historyID := uuid.FromStringOrNil(uriItems[5])
	if historyID == uuid.Nil {
		log.Errorf("Invalid history ID.")
		return simpleResponse(400), nil
	}

	tmp, err := h.aiprompthistoryHandler.Get(ctx, aiID, historyID)
	if err != nil {
		log.Debugf("Could not get item. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
