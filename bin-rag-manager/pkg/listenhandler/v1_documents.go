package listenhandler

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-rag-manager/models/document"
)

func (h *listenHandler) processV1DocumentsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1DocumentsGet",
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		log.Errorf("Could not parse URI. err: %v", err)
		return simpleResponse(400), nil
	}

	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
	if err != nil {
		log.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	filters, err := utilhandler.ConvertFilters[document.FieldStruct, document.Field](document.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	docs, err := h.ragHandler.DocumentList(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not list documents. err: %v", err)
		return simpleResponse(500), nil
	}

	return jsonResponse(200, docs), nil
}

func (h *listenHandler) processV1DocumentsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1DocumentsIDGet",
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Could not parse document ID from URI.")
		return simpleResponse(400), nil
	}

	d, err := h.ragHandler.DocumentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get document. err: %v", err)
		return simpleResponse(404), nil
	}

	return jsonResponse(200, d), nil
}
