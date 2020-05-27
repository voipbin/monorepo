package listenhandler

import (
	"encoding/json"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
)

// processV1ConferencesPost handles /v1/conferences request
func (h *listenHandler) processV1ConferencesPost(m *rabbitmq.Request) (*rabbitmq.Response, error) {

	type Data struct {
		Type conference.Type `json:"type"`
	}

	var data Data
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		return nil, err
	}

	cf, err := h.conferenceHandler.Start(data.Type, nil)
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
