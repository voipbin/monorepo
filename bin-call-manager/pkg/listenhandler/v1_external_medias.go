package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/pkg/listenhandler/models/request"
)

// processV1ExternalMediasGet handles GET /v1/external-medias request
func (h *listenHandler) processV1ExternalMediasGet(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ExternalMediasGet",
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

	tmps, err := h.externalMediaHandler.Gets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get external medias. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmps)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmps, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ExternalMediasPost handles POST /v1/external-medias request
func (h *listenHandler) processV1ExternalMediasPost(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ExternalMediasPost",
		"request": m,
	})

	var req request.V1DataExternalMediasPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.externalMediaHandler.Start(
		ctx,
		req.ReferenceType,
		req.ReferenceID,
		req.NoInsertMedia,
		req.ExternalHost,
		externalmedia.Encapsulation(req.Encapsulation),
		externalmedia.Transport(req.Transport),
		req.ConnectionType,
		req.Format,
		req.Direction,
	)
	if err != nil {
		log.Errorf("Could not start the external media. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ExternalMediasIDGet handles GET /v1/external-medias/<external-media-id> request
func (h *listenHandler) processV1ExternalMediasIDGet(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ExternalMediasIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.externalMediaHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get external media. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ExternalMediasIDDelete handles DELETE /v1/external-medias/<id> request
func (h *listenHandler) processV1ExternalMediasIDDelete(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ExternalMediasIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.externalMediaHandler.Stop(ctx, id)
	if err != nil {
		log.Errorf("Could not stop the extneral media. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
