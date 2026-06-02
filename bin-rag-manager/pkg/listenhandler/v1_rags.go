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
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-rag-manager/models/rag"
)

func (h *listenHandler) processV1RagsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1RagsPost",
	})

	var reqData struct {
		CustomerID     uuid.UUID   `json:"customer_id"`
		Name           string      `json:"name"`
		Description    string      `json:"description"`
		StorageFileIDs []uuid.UUID `json:"storage_file_ids"`
		SourceURLs     []string    `json:"source_urls"`
	}
	if err := json.Unmarshal(m.Data, &reqData); err != nil {
		log.Errorf("Could not unmarshal request. err: %v", err)
		return simpleResponse(400), nil
	}

	if reqData.CustomerID == uuid.Nil {
		log.Errorf("Customer ID is required.")
		return simpleResponse(400), nil
	}

	if reqData.Name == "" {
		log.Errorf("Name is required.")
		return simpleResponse(400), nil
	}

	if len(reqData.StorageFileIDs) == 0 && len(reqData.SourceURLs) == 0 {
		log.Errorf("At least one storage_file_ids or source_urls is required.")
		return simpleResponse(400), nil
	}

	r, err := h.ragHandler.RagCreate(ctx, reqData.CustomerID, reqData.Name, reqData.Description, reqData.StorageFileIDs, reqData.SourceURLs)
	if err != nil {
		log.Errorf("Could not create rag. err: %v", err)
		return errorResponse(err), nil
	}

	return jsonResponse(200, r), nil
}

func (h *listenHandler) processV1RagsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1RagsGet",
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

	filters, err := utilhandler.ConvertFilters[rag.FieldStruct, rag.Field](rag.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	rags, err := h.ragHandler.RagList(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not list rags. err: %v", err)
		return errorResponse(err), nil
	}

	return jsonResponse(200, rags), nil
}

func (h *listenHandler) processV1RagsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1RagsIDGet",
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Could not parse rag ID from URI.")
		return simpleResponse(400), nil
	}

	r, err := h.ragHandler.RagGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get rag. err: %v", err)
		return errorResponse(err), nil
	}

	return jsonResponse(200, r), nil
}

func (h *listenHandler) processV1RagsIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1RagsIDPut",
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Could not parse rag ID from URI.")
		return simpleResponse(400), nil
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal request. err: %v", err)
		return simpleResponse(400), nil
	}

	fields := make(map[rag.Field]any)
	if req.Name != nil {
		fields[rag.FieldName] = *req.Name
	}
	if req.Description != nil {
		fields[rag.FieldDescription] = *req.Description
	}

	if len(fields) == 0 {
		log.Errorf("No fields to update.")
		return simpleResponse(400), nil
	}

	r, err := h.ragHandler.RagUpdate(ctx, id, fields)
	if err != nil {
		log.Errorf("Could not update rag. err: %v", err)
		return errorResponse(err), nil
	}

	return jsonResponse(200, r), nil
}

func (h *listenHandler) processV1RagsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1RagsIDDelete",
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Could not parse rag ID from URI.")
		return simpleResponse(400), nil
	}

	if err := h.ragHandler.RagDelete(ctx, id); err != nil {
		log.Errorf("Could not delete rag. err: %v", err)
		return errorResponse(err), nil
	}

	return simpleResponse(200), nil
}

func (h *listenHandler) processV1RagsIDSourcesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1RagsIDSourcesPost",
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	ragID := uuid.FromStringOrNil(uriItems[3])
	if ragID == uuid.Nil {
		log.Errorf("Could not parse rag ID from URI.")
		return simpleResponse(400), nil
	}

	var reqData struct {
		StorageFileIDs []uuid.UUID `json:"storage_file_ids"`
		SourceURLs     []string    `json:"source_urls"`
	}

	if err := json.Unmarshal(m.Data, &reqData); err != nil {
		log.Errorf("Could not unmarshal request. err: %v", err)
		return simpleResponse(400), nil
	}

	if len(reqData.StorageFileIDs) == 0 && len(reqData.SourceURLs) == 0 {
		log.Errorf("At least one storage_file_ids or source_urls is required.")
		return simpleResponse(400), nil
	}

	r, err := h.ragHandler.RagAddSources(ctx, ragID, reqData.StorageFileIDs, reqData.SourceURLs)
	if err != nil {
		log.Errorf("Could not add sources. err: %v", err)
		return errorResponse(err), nil
	}

	return jsonResponse(200, r), nil
}

func (h *listenHandler) processV1RagsIDSourcesIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1RagsIDSourcesIDDelete",
	})

	// URI: /v1/rags/{rag-id}/sources/{source-id}
	// Split: ["", "v1", "rags", "{rag-id}", "sources", "{source-id}"]
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 6 {
		return simpleResponse(400), nil
	}

	ragID := uuid.FromStringOrNil(uriItems[3])
	if ragID == uuid.Nil {
		log.Errorf("Could not parse rag ID from URI.")
		return simpleResponse(400), nil
	}

	sourceID := uuid.FromStringOrNil(uriItems[5])
	if sourceID == uuid.Nil {
		log.Errorf("Could not parse source ID from URI.")
		return simpleResponse(400), nil
	}

	r, err := h.ragHandler.RagRemoveSource(ctx, ragID, sourceID)
	if err != nil {
		log.Errorf("Could not remove source. err: %v", err)
		return errorResponse(err), nil
	}

	return jsonResponse(200, r), nil
}
