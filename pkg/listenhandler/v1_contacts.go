package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processV1ContactsGet handles /v1/contacts GET request
func (h *listenHandler) processV1ContactsGet(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1ContactsGet",
	})

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// get endpoint
	endpoint, err := url.QueryUnescape(u.Query().Get("endpoint"))
	if err != nil {
		log.Errorf("Could not unescape the parameter. err: %v", err)
		return nil, err
	}

	resContacts, err := h.contactHandler.ContactGetsByEndpoint(ctx, endpoint)
	if err != nil {
		log.Errorf("Could not get contacts. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(resContacts)
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

// processV1ContactsPut handles /v1/contatcs PUT request
func (h *listenHandler) processV1ContactsPut(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1ContactsPut",
	})

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// get endpoint
	endpoint, err := url.QueryUnescape(u.Query().Get("endpoint"))
	if err != nil {
		log.Errorf("Could not unescape the parameter. err: %v", err)
		return nil, err
	}

	if err := h.contactHandler.ContactRefreshByEndpoint(ctx, endpoint); err != nil {
		log.Errorf("Could not refresh the contact info. endpoint: %s, err: %v", endpoint, err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
	}

	return res, nil
}
