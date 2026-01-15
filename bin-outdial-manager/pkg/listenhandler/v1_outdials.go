package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-outdial-manager/models/outdial"
	"monorepo/bin-outdial-manager/pkg/listenhandler/models/request"
)

// v1OutdialsPost handles /v1/outdials POST request
func (h *listenHandler) v1OutdialsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1OutdialsPost",
		},
	)
	log.WithField("request", m).Debug("Executing v1OutdialsPost.")

	var req request.V1DataOutdialsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// create
	tmp, err := h.outdialHandler.Create(
		ctx,
		req.CustomerID,
		req.CampaignID,
		req.Name,
		req.Detail,
		req.Data,
	)
	if err != nil {
		log.Errorf("Could not create outdial. err: %v", err)
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

// v1OutdialsGet handles /v1/outdials GET request
func (h *listenHandler) v1OutdialsGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1OutdialsGet",
		"request": req,
	})

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params from URI
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// Parse filters from request data (body)
	var filters map[string]any
	if len(req.Data) > 0 {
		if err := json.Unmarshal(req.Data, &filters); err != nil {
			log.Errorf("Could not unmarshal filters. err: %v", err)
			return nil, fmt.Errorf("could not unmarshal filters: %w", err)
		}
	}

	log.WithFields(logrus.Fields{
		"filters":          filters,
		"filters_raw_data": string(req.Data),
	}).Debug("v1OutdialsGet: Parsed filters from request body")

	// Convert string map to typed field map
	typedFilters, err := outdial.ConvertStringMapToFieldMap(filters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return nil, fmt.Errorf("could not convert filters: %w", err)
	}

	log.WithFields(logrus.Fields{
		"typed_filters": typedFilters,
	}).Debug("v1OutdialsGet: Converted filters to typed field map (check UUID types)")

	tmp, err := h.outdialHandler.Gets(ctx, pageToken, pageSize, typedFilters)
	if err != nil {
		logrus.Errorf("Could not get outdials. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		logrus.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1OutdialsIDGet handles /v1/outdials/<outdial-id> GET request
func (h *listenHandler) v1OutdialsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "v1OutdialsIDGet",
			"outdial_id": id,
		},
	)
	log.WithField("request", m).Debug("Executing v1OutdialsIDGet.")

	tmp, err := h.outdialHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get outdial. err: %v", err)
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

// v1OutdialsIDDelete handles /v1/outdials/<outdial-id> DELETE request
func (h *listenHandler) v1OutdialsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "v1OutdialsIDDelete",
			"outdial_id": id,
		},
	)
	log.WithField("request", m).Debug("Executing v1OutdialsIDDelete.")

	tmp, err := h.outdialHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete outdial. err: %v", err)
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

// v1OutdialsIDPut handles /v1/outdials/<outdial-id> PUT request
func (h *listenHandler) v1OutdialsIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	log := logrus.WithFields(
		logrus.Fields{
			"func":       "v1OutdialsIDPut",
			"outdial_id": id,
		},
	)
	log.WithField("request", m).Debug("Executing v1OutdialsIDPut.")

	var req request.V1DataOutdialsIDPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// handle
	tmp, err := h.outdialHandler.UpdateBasicInfo(ctx, id, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not update outdial. err: %v", err)
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

// v1OutdialsIDCampaignIDPut handles /v1/outdials/<outdial-id>/campaign_id PUT request
func (h *listenHandler) v1OutdialsIDCampaignIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	log := logrus.WithFields(
		logrus.Fields{
			"func":       "v1OutdialsIDCampaignIDPut",
			"outdial_id": id,
		},
	)
	log.WithField("request", m).Debug("Executing v1OutdialsIDCampaignIDPut.")

	var req request.V1DataOutdialsIDCampaignIDPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// handle
	tmp, err := h.outdialHandler.UpdateCampaignID(ctx, id, req.CampaignID)
	if err != nil {
		log.Errorf("Could not update outdial. err: %v", err)
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

// v1OutdialsIDDataPut handles /v1/outdials/<outdial-id>/data PUT request
func (h *listenHandler) v1OutdialsIDDataPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	log := logrus.WithFields(
		logrus.Fields{
			"func":       "v1OutdialsIDDataPut",
			"outdial_id": id,
		},
	)
	log.WithField("request", m).Debug("Executing v1OutdialsIDDataPut.")

	var req request.V1DataOutdialsIDDataPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// handle
	tmp, err := h.outdialHandler.UpdateData(ctx, id, req.Data)
	if err != nil {
		log.Errorf("Could not update outdial. err: %v", err)
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

// v1OutdialsIDAvailableGet handles /v1/outdials/<outdial-id>/available GET request
func (h *listenHandler) v1OutdialsIDAvailableGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	log := logrus.WithFields(logrus.Fields{
		"func":       "v1OutdialsIDAvailableGet",
		"outdial_id": id,
		"request":    m,
	})

	// Parse filters from request data (body)
	var filters map[string]any
	if len(m.Data) > 0 {
		if err := json.Unmarshal(m.Data, &filters); err != nil {
			log.Errorf("Could not unmarshal filters. err: %v", err)
			return nil, fmt.Errorf("could not unmarshal filters: %w", err)
		}
	}

	log.WithFields(logrus.Fields{
		"filters":          filters,
		"filters_raw_data": string(m.Data),
	}).Debug("v1OutdialsIDAvailableGet: Parsed filters from request body")

	// Extract parameters from filters
	tryCount0, tryCount1, tryCount2, tryCount3, tryCount4 := 0, 0, 0, 0, 0
	limit := uint64(0)

	if tc0, ok := filters["try_count_0"]; ok {
		if tc0Float, ok := tc0.(float64); ok {
			tryCount0 = int(tc0Float)
		}
	}
	if tc1, ok := filters["try_count_1"]; ok {
		if tc1Float, ok := tc1.(float64); ok {
			tryCount1 = int(tc1Float)
		}
	}
	if tc2, ok := filters["try_count_2"]; ok {
		if tc2Float, ok := tc2.(float64); ok {
			tryCount2 = int(tc2Float)
		}
	}
	if tc3, ok := filters["try_count_3"]; ok {
		if tc3Float, ok := tc3.(float64); ok {
			tryCount3 = int(tc3Float)
		}
	}
	if tc4, ok := filters["try_count_4"]; ok {
		if tc4Float, ok := tc4.(float64); ok {
			tryCount4 = int(tc4Float)
		}
	}
	if lim, ok := filters["limit"]; ok {
		if limFloat, ok := lim.(float64); ok {
			limit = uint64(limFloat)
		}
	}

	tmp, err := h.outdialTargetHandler.GetAvailable(ctx, id, tryCount0, tryCount1, tryCount2, tryCount3, tryCount4, limit)
	if err != nil {
		log.Errorf("Could not get available targets. err: %v", err)
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

// v1OutdialsIDTargetsPost handles /v1/outdials/<outdial-id>/targets POST request
func (h *listenHandler) v1OutdialsIDTargetsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	log := logrus.WithFields(
		logrus.Fields{
			"func":       "v1OutdialsIDTargetsPost",
			"outdial_id": id,
		},
	)
	log.WithField("request", m).Debug("Executing v1OutdialsIDTargetsPost.")

	var req request.V1DataOutdialsIDTargetsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// create
	tmp, err := h.outdialTargetHandler.Create(
		ctx,
		id,
		req.Name,
		req.Detail,
		req.Data,
		req.Destination0,
		req.Destination1,
		req.Destination2,
		req.Destination3,
		req.Destination4,
	)
	if err != nil {
		log.Errorf("Could not create outdialtarget. err: %v", err)
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

// v1OutdialsIDTargetsGet handles /v1/outdials/<outdial-id>/targets GET request
func (h *listenHandler) v1OutdialsIDTargetsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	log := logrus.WithFields(
		logrus.Fields{
			"func":       "v1OutdialsIDTargetsGet",
			"outdial_id": id,
		},
	)
	log.WithField("request", m).Debug("Executing v1OutdialsIDTargetsGet.")

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	tmp, err := h.outdialTargetHandler.GetsByOutdialID(ctx, id, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not get outdialtargets. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		logrus.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
