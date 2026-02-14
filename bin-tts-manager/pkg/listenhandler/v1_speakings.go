package listenhandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-tts-manager/models/speaking"
	"monorepo/bin-tts-manager/pkg/listenhandler/models/request"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// v1SpeakingsPost handles /v1/speakings POST request
// It creates a new speaking session based on the provided request data.
func (h *listenHandler) v1SpeakingsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1SpeakingsPost",
	})

	var req request.V1DataSpeakingsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return nil, err
	}
	log.WithField("request", req).Debugf("Processing v1SpeakingsPost.")

	tmp, err := h.speakingHandler.Create(ctx, req.CustomerID, req.ReferenceType, req.ReferenceID, req.Language, req.Provider, req.VoiceID, req.Direction)
	if err != nil {
		log.Errorf("Could not create a speaking. err: %v", err)
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

// v1SpeakingsGet handles /v1/speakings GET request
// It retrieves a list of speakings with optional filtering.
func (h *listenHandler) v1SpeakingsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1SpeakingsGet",
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		log.Errorf("Could not parse URI. err: %v", err)
		return nil, err
	}

	q := u.Query()
	pageToken := q.Get("page_token")
	pageSize := uint64(100) // default page size
	if v := q.Get("page_size"); v != "" {
		if parsed, err := strconv.ParseUint(v, 10, 64); err == nil {
			pageSize = parsed
		}
	}

	// Build filters
	filters := map[speaking.Field]any{
		speaking.FieldDeleted: false,
	}

	if v := q.Get("customer_id"); v != "" {
		filters[speaking.FieldCustomerID] = uuid.FromStringOrNil(v)
	}
	if v := q.Get("reference_type"); v != "" {
		filters[speaking.FieldReferenceType] = v
	}
	if v := q.Get("reference_id"); v != "" {
		filters[speaking.FieldReferenceID] = uuid.FromStringOrNil(v)
	}
	if v := q.Get("status"); v != "" {
		filters[speaking.FieldStatus] = v
	}

	log.WithField("filters", filters).Debugf("Processing v1SpeakingsGet.")

	tmp, err := h.speakingHandler.Gets(ctx, pageToken, pageSize, filters)
	if err != nil {
		log.Errorf("Could not get speakings. err: %v", err)
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

// v1SpeakingsIDGet handles /v1/speakings/{id} GET request
func (h *listenHandler) v1SpeakingsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1SpeakingsIDGet",
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/speakings/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	speakingID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.speakingHandler.Get(ctx, speakingID)
	if err != nil {
		log.Errorf("Could not get the speaking. err: %v", err)
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

// v1SpeakingsIDSayPost handles /v1/speakings/{id}/say POST request
func (h *listenHandler) v1SpeakingsIDSayPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1SpeakingsIDSayPost",
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/speakings/a6f4eae8-8a74-11ea-af75-3f1e61b9a236/say"
	tmpVals := strings.Split(u.Path, "/")
	speakingID := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataSpeakingsIDSayPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return nil, err
	}
	log.WithField("request", req).Debugf("Processing v1SpeakingsIDSayPost. speaking_id: %s", speakingID)

	tmp, err := h.speakingHandler.Say(ctx, speakingID, req.Text)
	if err != nil {
		log.Errorf("Could not say text to speaking. err: %v", err)
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

// v1SpeakingsIDFlushPost handles /v1/speakings/{id}/flush POST request
func (h *listenHandler) v1SpeakingsIDFlushPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1SpeakingsIDFlushPost",
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/speakings/a6f4eae8-8a74-11ea-af75-3f1e61b9a236/flush"
	tmpVals := strings.Split(u.Path, "/")
	speakingID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.speakingHandler.Flush(ctx, speakingID)
	if err != nil {
		log.Errorf("Could not flush the speaking. err: %v", err)
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

// v1SpeakingsIDStopPost handles /v1/speakings/{id}/stop POST request
func (h *listenHandler) v1SpeakingsIDStopPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1SpeakingsIDStopPost",
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/speakings/a6f4eae8-8a74-11ea-af75-3f1e61b9a236/stop"
	tmpVals := strings.Split(u.Path, "/")
	speakingID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.speakingHandler.Stop(ctx, speakingID)
	if err != nil {
		log.Errorf("Could not stop the speaking. err: %v", err)
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

// v1SpeakingsIDDelete handles /v1/speakings/{id} DELETE request
func (h *listenHandler) v1SpeakingsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1SpeakingsIDDelete",
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/speakings/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	speakingID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.speakingHandler.Delete(ctx, speakingID)
	if err != nil {
		log.Errorf("Could not delete the speaking. err: %v", err)
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
