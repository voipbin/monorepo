package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonsock "monorepo/bin-common-handler/models/sock"
	"monorepo/bin-talk-manager/models/pin"
	"monorepo/bin-talk-manager/pkg/listenhandler/models/request"
)

// parseOptionalUUID parses a UUID string that may be empty. It returns false for a non-empty
// string that is not a UUID.
func parseOptionalUUID(s string) (uuid.UUID, bool) {
	if s == "" {
		return uuid.Nil, true
	}
	id := uuid.FromStringOrNil(s)
	return id, id != uuid.Nil
}

func jsonResponse(v any) (*commonsock.Response, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return &commonsock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) v1PinsPost(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	log := logrus.WithField("func", "v1PinsPost")

	var req request.V1DataPinsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		return simpleResponse(400), nil
	}

	customerID, ok1 := parseOptionalUUID(req.CustomerID)
	ownerID, ok2 := parseOptionalUUID(req.OwnerID)
	targetID, ok3 := parseOptionalUUID(req.TargetID)
	if !ok1 || !ok2 || !ok3 {
		log.Errorf("Could not parse the ids of the request.")
		return simpleResponse(400), nil
	}

	res, err := h.pinHandler.PinCreate(ctx, customerID, req.OwnerType, ownerID, pin.TargetType(req.TargetType), targetID)
	if err != nil {
		log.Errorf("Could not create the pin. err: %v", err)
		return errorResponse(err), nil
	}

	return jsonResponse(res)
}

func (h *listenHandler) v1PinsGet(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	log := logrus.WithField("func", "v1PinsGet")

	u, err := url.Parse(m.URI)
	if err != nil {
		log.Errorf("Could not parse the uri. err: %v", err)
		return simpleResponse(400), nil
	}
	tmpSize, _ := strconv.Atoi(u.Query().Get("page_size"))
	pageSize := uint64(0)
	if tmpSize > 0 {
		pageSize = uint64(tmpSize)
	}
	pageToken := u.Query().Get("page_token")

	var req request.V1DataPinsGet
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		return simpleResponse(400), nil
	}

	customerID, ok1 := parseOptionalUUID(req.CustomerID)
	ownerID, ok2 := parseOptionalUUID(req.OwnerID)
	chatID, ok3 := parseOptionalUUID(req.ChatID)
	if !ok1 || !ok2 || !ok3 {
		log.Errorf("Could not parse the ids of the request.")
		return simpleResponse(400), nil
	}

	res, err := h.pinHandler.PinList(ctx, customerID, req.OwnerType, ownerID, pin.TargetType(req.TargetType), chatID, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not list the pins. err: %v", err)
		return errorResponse(err), nil
	}

	return jsonResponse(res)
}

func (h *listenHandler) v1PinsIDDelete(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	log := logrus.WithField("func", "v1PinsIDDelete")

	matches := regV1PinsID.FindStringSubmatch(m.URI)
	id := uuid.FromStringOrNil(matches[1])
	if id == uuid.Nil {
		return simpleResponse(400), nil
	}

	var req request.V1DataPinsIDDelete
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		return simpleResponse(400), nil
	}

	customerID, ok1 := parseOptionalUUID(req.CustomerID)
	ownerID, ok2 := parseOptionalUUID(req.OwnerID)
	if !ok1 || !ok2 {
		log.Errorf("Could not parse the ids of the request.")
		return simpleResponse(400), nil
	}

	res, err := h.pinHandler.PinDelete(ctx, id, customerID, req.OwnerType, ownerID)
	if err != nil {
		log.Errorf("Could not delete the pin. err: %v", err)
		return errorResponse(err), nil
	}

	return jsonResponse(res)
}
