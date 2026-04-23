package listenhandler

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-route-manager/pkg/listenhandler/models/request"
	"monorepo/bin-route-manager/pkg/telnyxclient"
)

// v1ProvidersSetupPost handles POST /v1/providers/setup.
// Returns 422 for invalid Telnyx API keys, 400 for other errors, 200 on success.
func (h *listenHandler) v1ProvidersSetupPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "v1ProvidersSetupPost")
	log.WithField("request", m).Debug("Received request.")

	var req request.V1DataProvidersSetupPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal request. err: %v", err)
		return nil, err
	}

	res, err := h.providerHandler.Setup(ctx, req.Carrier, req.Name, req.Detail, req.Credentials.APIKey)
	if err != nil {
		if errors.Is(err, telnyxclient.ErrInvalidKey) {
			return simpleResponse(422), nil
		}
		log.Errorf("Could not set up provider. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(res)
	if err != nil {
		log.Errorf("Could not marshal response. err: %v", err)
		return nil, err
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}
