package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/pkg/listenhandler/models/request"
)

// processV1HooksGet handles GET /v1/hooks (Meta hub challenge verification).
func (h *listenHandler) processV1HooksGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1HooksGet",
		"request": m,
	})

	var req request.V1DataHooksPost // reuses the same model — embeds hmhook.Hook fields
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Debugf("Could not unmarshal data. err: %v", err)
		return simpleResponse(400), nil
	}

	// hub.* params are in req.ReceviedURI (the forwarded external URL),
	// NOT in m.URI (the internal RPC path "/v1/hooks").
	u, err := url.Parse(req.ReceviedURI)
	if err != nil {
		log.Debugf("Could not parse ReceviedURI. err: %v", err)
		return simpleResponse(400), nil
	}

	q := u.Query()
	mode := q.Get("hub.mode")
	token := q.Get("hub.verify_token")
	challenge := q.Get("hub.challenge")

	result, err := h.conversationHandler.HookVerify(ctx, req.ReceviedURI, mode, token, challenge)
	if err != nil {
		log.Errorf("HookVerify failed. err: %v", err)
		return simpleResponse(403), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "text/plain",
		Data:       []byte(result),
	}, nil
}

// processV1HooksPost handles
// POST /v1/hooks request
func (h *listenHandler) processV1HooksPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1HooksPost",
		"request": m,
	})

	var req request.V1DataHooksPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log.WithField("request", req).Debugf("Received hook request. request_uri: %s", req.ReceviedURI)

	if errHook := h.conversationHandler.Hook(ctx, req.ReceviedURI, req.ReceivedMethod, req.ReceivedSignature, req.ReceivedData); errHook != nil {
		log.Errorf("Could not hook the message correctly. err: %v", errHook)
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}
