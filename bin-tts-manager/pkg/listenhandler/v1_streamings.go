package listenhandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-tts-manager/pkg/listenhandler/models/request"
	"net/url"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// v1StreamingsPost handles /v1/streamings POST request
// It starts a new streaming session based on the provided request data.
func (h *listenHandler) v1StreamingsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1StreamingsPost",
	})

	var req request.V1DataStreamingsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return nil, err
	}
	log.WithField("request", req).Debugf("Processing v1StreamingsPost.")

	tmp, err := h.streamingHandler.Start(ctx, req.CustomerID, req.ActiveflowID, req.ReferenceType, req.ReferenceID, req.Language, req.Gender, req.Direction)
	if err != nil {
		log.Errorf("Could not create a streaming. err: %v", err)
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

// v1StreamingsIDDelete handles /v1/streamings/<id> DELETE request
func (h *listenHandler) v1StreamingsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1StreamingsIDDelete",
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/streamings/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	streamingID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.streamingHandler.Stop(ctx, streamingID)
	if err != nil {
		log.Errorf("Could not delete the streaming. err: %v", err)
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

// v1StreamingsIDSayInitPost handles /v1/streamings/<id>/say_init POST request
func (h *listenHandler) v1StreamingsIDSayInitPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1StreamingsIDSayInitPost",
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/streamings/a6f4eae8-8a74-11ea-af75-3f1e61b9a236/say"
	tmpVals := strings.Split(u.Path, "/")
	streamingID := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataStreamingsIDSayInitPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return nil, err
	}
	log.WithField("request", req).Debugf("Processing v1StreamingsIDSayInitPost. streaming_id: %s", streamingID)

	tmp, errSay := h.streamingHandler.SayInit(ctx, streamingID, req.MessageID)
	if errSay != nil {
		log.Errorf("Could not add to the say streaming. err: %v", errSay)
		return nil, errSay
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

// v1StreamingsIDSayAddPost handles /v1/streamings/<id>/say_add POST request
func (h *listenHandler) v1StreamingsIDSayAddPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1StreamingsIDSayAddPost",
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/streamings/a6f4eae8-8a74-11ea-af75-3f1e61b9a236/say"
	tmpVals := strings.Split(u.Path, "/")
	streamingID := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataStreamingsIDSayAddPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return nil, err
	}
	log.WithField("request", req).Debugf("Processing v1StreamingsIDSayAddPost. streaming_id: %s", streamingID)

	if errSay := h.streamingHandler.SayAdd(ctx, streamingID, req.MessageID, req.Text); errSay != nil {
		log.Errorf("Could not add to the say streaming. err: %v", errSay)
		return nil, errSay
	}

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}

// v1StreamingsIDSayStopPost handles /v1/streamings/<id>/say_stop POST request
func (h *listenHandler) v1StreamingsIDSayStopPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1StreamingsIDSayStopPost",
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/streamings/a6f4eae8-8a74-11ea-af75-3f1e61b9a236/say_stop"
	tmpVals := strings.Split(u.Path, "/")
	streamingID := uuid.FromStringOrNil(tmpVals[3])

	if errStop := h.streamingHandler.SayStop(ctx, streamingID); errStop != nil {
		log.Errorf("Could not stop the say streaming. err: %v", errStop)
		return nil, errStop
	}

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}

// v1StreamingsIDMessageFinishPost handles /v1/streamings/<id>/message_finish POST request
func (h *listenHandler) v1StreamingsIDMessageFinishPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1StreamingsIDMessageFinishPost",
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/streamings/a6f4eae8-8a74-11ea-af75-3f1e61b9a236/say"
	tmpVals := strings.Split(u.Path, "/")
	streamingID := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataStreamingsIDSayFinishPost
	if errUnmarshal := json.Unmarshal(m.Data, &req); errUnmarshal != nil {
		log.Errorf("Could not unmarshal the data. err: %v", errUnmarshal)
		return nil, errUnmarshal
	}
	log.WithField("request", req).Debugf("Processing v1StreamingsIDMessageFinishPost. streaming_id: %s", streamingID)

	tmp, err := h.streamingHandler.SayFinish(ctx, streamingID, req.MessageID)
	if err != nil {
		log.Errorf("Could not add to the say streaming. err: %v", err)
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
