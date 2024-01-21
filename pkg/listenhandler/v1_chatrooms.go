package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// v1ChatroomsGet handles /v1/chatrooms GET request
func (h *listenHandler) v1ChatroomsGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1ChatroomsGet",
	})
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get owner_id
	ownerID := uuid.FromStringOrNil(u.Query().Get("owner_id"))

	// get filters
	filters := getFilters(u)

	tmp, err := h.chatroomHandler.GetsByOwnerID(ctx, ownerID, pageToken, pageSize, filters)
	if err != nil {
		log.Errorf("Could not get chats by GetsByOwnerID. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ChatroomsIDGet handles /v1/chatrooms/{id} GET request
func (h *listenHandler) v1ChatroomsIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1ChatroomsIDGet",
	})
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	chatroomID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.chatroomHandler.Get(ctx, chatroomID)
	if err != nil {
		log.Errorf("Could not get chatroom info. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ChatroomsIDDelete handles /v1/chatrooms/{id} Delete request
func (h *listenHandler) v1ChatroomsIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1ChatroomsIDDelete",
	})
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/chatrooms/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])
	log = log.WithField("chatroom_id", id)

	tmp, err := h.chatroomHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the chatroom. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
