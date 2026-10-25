package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/pkg/listenhandler/models/request"
)

// All the handlers of this file log logM, the copy of the request whose body is redacted: the
// body of a secret request must never reach a log.

// providerNameFromURI returns the provider name of the matched URI.
func providerNameFromURI(m []string) number.ProviderName {
	return number.ProviderName(m[1])
}

func jsonResponse(v any, logM *sock.Request) *sock.Response {
	data, err := json.Marshal(v)
	if err != nil {
		logrus.WithField("request", logM).Errorf("Could not marshal the response message. err: %v", err)
		return simpleResponse(500)
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}
}

// processV1NumberProvidersGet handles GET /v1/number_providers request
func (h *listenHandler) processV1NumberProvidersGet(ctx context.Context, logM *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1NumberProvidersGet",
		"request": logM,
	})

	res, err := h.providerHandler.List(ctx)
	if err != nil {
		log.Errorf("Could not list the providers. err: %v", err)
		return errorResponse(err), nil
	}

	return jsonResponse(res, logM), nil
}

// processV1NumberProvidersIDGet handles GET /v1/number_providers/<name> request
func (h *listenHandler) processV1NumberProvidersIDGet(ctx context.Context, logM *sock.Request, matches []string) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1NumberProvidersIDGet",
		"request": logM,
	})

	res, err := h.providerHandler.Get(ctx, providerNameFromURI(matches))
	if err != nil {
		log.Errorf("Could not get the provider. err: %v", err)
		return errorResponse(err), nil
	}

	return jsonResponse(res, logM), nil
}

// processV1NumberProvidersIDPut handles PUT /v1/number_providers/<name> request
func (h *listenHandler) processV1NumberProvidersIDPut(ctx context.Context, m *sock.Request, logM *sock.Request, matches []string) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1NumberProvidersIDPut",
		"request": logM,
	})

	var req request.V1DataNumberProvidersIDPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return simpleResponse(400), nil
	}

	res, err := h.providerHandler.Update(ctx, providerNameFromURI(matches), req.ActorID, req.Enabled, req.Priority, req.Config)
	if err != nil {
		log.Errorf("Could not update the provider. err: %v", err)
		return errorResponse(err), nil
	}

	return jsonResponse(res, logM), nil
}

// processV1NumberProvidersIDSecretPut handles PUT /v1/number_providers/<name>/secret request.
// m holds the secret and is used only to read it; everything that is logged comes from logM.
func (h *listenHandler) processV1NumberProvidersIDSecretPut(ctx context.Context, m *sock.Request, logM *sock.Request, matches []string) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1NumberProvidersIDSecretPut",
		"request": logM,
	})

	var req request.V1DataNumberProvidersIDSecretPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		// the message of the decoder may quote the input, so only the type of the error is logged
		log.Errorf("Could not unmarshal the data. err_type: %T", err)
		return simpleResponse(400), nil
	}

	res, err := h.providerHandler.UpdateSecret(ctx, providerNameFromURI(matches), req.ActorID, req.Secret)
	if err != nil {
		log.Errorf("Could not update the provider secret. err: %v", err)
		return errorResponse(err), nil
	}

	return jsonResponse(res, logM), nil
}

// processV1NumberProvidersIDSecretDelete handles DELETE /v1/number_providers/<name>/secret request
func (h *listenHandler) processV1NumberProvidersIDSecretDelete(ctx context.Context, m *sock.Request, logM *sock.Request, matches []string) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1NumberProvidersIDSecretDelete",
		"request": logM,
	})

	var req request.V1DataNumberProvidersIDSecretDelete
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return simpleResponse(400), nil
	}

	res, err := h.providerHandler.DeleteSecret(ctx, providerNameFromURI(matches), req.ActorID)
	if err != nil {
		log.Errorf("Could not delete the provider secret. err: %v", err)
		return errorResponse(err), nil
	}

	return jsonResponse(res, logM), nil
}
