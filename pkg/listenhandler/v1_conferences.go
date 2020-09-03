package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/listenhandler/models/request"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
)

// processV1ConferencesPost handles /v1/conferences request
func (h *listenHandler) processV1ConferencesPost(m *rabbitmq.Request) (*rabbitmq.Response, error) {

	type Data struct {
		Type    conference.Type        `json:"type"`
		Name    string                 `json:"name"`
		Detail  string                 `json:"detail"`
		Timeout int                    `json:"timeout"`
		Data    map[string]interface{} `json:"data"`
	}

	var data Data
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		return nil, err
	}

	// create a request conference
	reqConf := &conference.Conference{
		Type:    data.Type,
		Name:    data.Name,
		Detail:  data.Detail,
		Timeout: data.Timeout,
		Data:    data.Data,
	}

	// create a conference
	cf, err := h.conferenceHandler.Start(reqConf, nil)
	if err != nil {
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(cf)
	if err != nil {
		return simpleResponse(400), nil
	}

	res := &rabbitmq.Response{
		StatusCode: 200,
		Data:       string(tmp),
	}

	return res, nil
}

// processV1ConferencesIDDelete handles /v1/conferences/<id> DELETE request
func (h *listenHandler) processV1ConferencesIDDelete(m *rabbitmq.Request) (*rabbitmq.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var data request.V1DataConferencesIDDelete
	if m.Data != "" {
		if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
			return nil, err
		}
	} else {
		data.Reason = ""
	}

	if err := h.conferenceHandler.Terminate(id, data.Reason); err != nil {
		return simpleResponse(400), nil
	}

	return simpleResponse(200), nil
}

// processV1ConferencesIDGet handles /v1/conferences/<id> GET request
func (h *listenHandler) processV1ConferencesIDGet(m *rabbitmq.Request) (*rabbitmq.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	cf, err := h.db.ConferenceGet(context.Background(), id)
	if err != nil {
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(cf)
	if err != nil {
		return simpleResponse(400), nil
	}

	res := &rabbitmq.Response{
		StatusCode: 200,
		Data:       string(tmp),
	}

	return res, nil
}

// processV1ConferencesIDCallsIDDelete handles /v1/conferences/<id>/calls/<id> DELETE request
func (h *listenHandler) processV1ConferencesIDCallsIDDelete(m *rabbitmq.Request) (*rabbitmq.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 6 {
		return simpleResponse(400), nil
	}
	cfID := uuid.FromStringOrNil(uriItems[3])
	callID := uuid.FromStringOrNil(uriItems[5])

	if cfID == uuid.Nil || callID == uuid.Nil {
		return simpleResponse(400), nil
	}

	if err := h.conferenceHandler.Leave(cfID, callID); err != nil {
		return simpleResponse(400), nil
	}

	return simpleResponse(200), nil
}
