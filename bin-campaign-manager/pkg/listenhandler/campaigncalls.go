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

	"monorepo/bin-campaign-manager/models/campaigncall"
)

// v1CampaigncallsGet handles /v1/campaigncalls GET request
func (h *listenHandler) v1CampaigncallsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1CampaigncallsGet",
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

	customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))
	campaignID := uuid.FromStringOrNil(u.Query().Get("campaign_id"))
	log.WithField("request", m).Debugf("Received request. customer_id: %s, campaign_id: %s", customerID, campaignID)

	var tmp []*campaigncall.Campaigncall
	if customerID != uuid.Nil {
		tmp, err = h.campaigncallHandler.GetsByCustomerID(ctx, customerID, pageToken, pageSize)
	} else {
		tmp, err = h.campaigncallHandler.GetsByCampaignID(ctx, campaignID, pageToken, pageSize)
	}
	if err != nil {
		log.Errorf("Could not get campaigns. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1CampaigncallsIDGet handles /v1/campaigncalls/{id} GET request
func (h *listenHandler) v1CampaigncallsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1CampaigncallsIDGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.campaigncallHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign info. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1CampaigncallsIDDelete handles /v1/campaigncalls/{id} DELETE request
func (h *listenHandler) v1CampaigncallsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1CampaigncallsIDDelete",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.campaigncallHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign info. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
