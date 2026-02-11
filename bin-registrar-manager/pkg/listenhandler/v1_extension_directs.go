package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"
)

// processV1ExtensionDirectsGet handles /v1/extension-directs?hash=<hash> GET request
func (h *listenHandler) processV1ExtensionDirectsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ExtensionDirectsGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	hash := u.Query().Get("hash")
	if hash == "" {
		log.Debugf("Missing hash parameter")
		return simpleResponse(400), nil
	}

	direct, err := h.extensionHandler.GetDirectByHash(ctx, hash)
	if err != nil {
		log.Errorf("Could not get extension direct by hash. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(direct)
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
