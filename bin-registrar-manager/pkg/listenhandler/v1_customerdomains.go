package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"
)

// processV1CustomerDomainsRealmGet handles /v1/customer_domains/realm/<realm> GET request
func (h *listenHandler) processV1CustomerDomainsRealmGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1CustomerDomainsRealmGet",
	})

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/customer_domains/realm/ab12.reg.voipbin.net"
	tmpVals := strings.Split(u.Path, "/")
	if len(tmpVals) < 5 || tmpVals[4] == "" {
		log.Errorf("Invalid realm.")
		return simpleResponse(400), nil
	}
	realm := tmpVals[4]

	tmp, err := h.customerDomainHandler.GetByRealm(ctx, realm)
	if err != nil {
		log.Errorf("Could not get customer domain info. realm: %s, err: %v", realm, err)
		return errorResponse(err), nil
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
