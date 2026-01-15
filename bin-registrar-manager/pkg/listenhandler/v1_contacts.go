package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-registrar-manager/pkg/listenhandler/models/request"

	"github.com/sirupsen/logrus"
)

// processV1ContactsGet handles /v1/contacts GET request
func (h *listenHandler) processV1ContactsGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ContactsGet",
		"request": req,
	})

	// Parse filters from request body
	var reqData request.V1DataContactsGet
	if len(req.Data) > 0 {
		if err := json.Unmarshal(req.Data, &reqData); err != nil {
			log.Errorf("Could not unmarshal filters. err: %v", err)
			return nil, err
		}
	}

	log.WithFields(logrus.Fields{
		"customer_id":      reqData.CustomerID,
		"extension":        reqData.Extension,
		"filters_raw_data": string(req.Data),
	}).Debug("processV1ContactsGet: Parsed filters from request body")

	resContacts, err := h.contactHandler.ContactGetsByExtension(ctx, reqData.CustomerID, reqData.Extension)
	if err != nil {
		log.Errorf("Could not get contacts. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(resContacts)
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

// processV1ContactsPut handles /v1/contatcs PUT request
func (h *listenHandler) processV1ContactsPut(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ContactsPut",
		"request": req,
	})

	// Parse filters from request body
	var reqData request.V1DataContactsPut
	if len(req.Data) > 0 {
		if err := json.Unmarshal(req.Data, &reqData); err != nil {
			log.Errorf("Could not unmarshal filters. err: %v", err)
			return nil, err
		}
	}

	log.WithFields(logrus.Fields{
		"customer_id":      reqData.CustomerID,
		"extension":        reqData.Extension,
		"filters_raw_data": string(req.Data),
	}).Debug("processV1ContactsPut: Parsed filters from request body")

	if err := h.contactHandler.ContactRefreshByEndpoint(ctx, reqData.CustomerID, reqData.Extension); err != nil {
		log.Errorf("Could not refresh the contact info. customer_id: %s, extension: %s, err: %v", reqData.CustomerID, reqData.Extension, err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
	}

	return res, nil
}
