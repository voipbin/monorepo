package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-contact-manager/pkg/listenhandler/models/request"
)

// parseStarIDs reads the contact id and the agent id out of
// /v1/contacts/{id}/stars/{agent_id}.
func parseStarIDs(uri string) (contactID, agentID uuid.UUID, ok bool) {
	u, err := url.Parse(uri)
	if err != nil {
		return uuid.Nil, uuid.Nil, false
	}

	// "", "v1", "contacts", "<contact>", "stars", "<agent>"
	parts := strings.Split(u.Path, "/")
	if len(parts) < 6 {
		return uuid.Nil, uuid.Nil, false
	}

	contactID = uuid.FromStringOrNil(parts[3])
	agentID = uuid.FromStringOrNil(parts[5])
	if contactID == uuid.Nil || agentID == uuid.Nil {
		return uuid.Nil, uuid.Nil, false
	}

	return contactID, agentID, true
}

// parseStarPaging reads page_size and page_token. A missing or invalid page_size is 0; the only
// caller, api-manager, always sends a page_size between 1 and 100.
func parseStarPaging(uri string) (uint64, string, bool) {
	u, err := url.Parse(uri)
	if err != nil {
		return 0, "", false
	}

	q := u.Query()
	size := uint64(0)
	if tmp, _ := strconv.Atoi(q.Get(PageSize)); tmp > 0 {
		size = uint64(tmp)
	}

	return size, q.Get(PageToken), true
}

// processV1ContactsIDStarsPost handles POST /v1/contacts/{id}/stars/{agent_id}:
// the agent stars the contact. Idempotent.
func (h *listenHandler) processV1ContactsIDStarsPost(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1ContactsIDStarsPost"})
	log.WithField("request", req).Debug("Received request.")

	contactID, agentID, ok := parseStarIDs(req.URI)
	if !ok {
		return simpleResponse(400), nil
	}

	var body request.V1DataContactsIDStars
	if len(req.Data) > 0 {
		_ = json.Unmarshal(req.Data, &body)
	}
	if body.CustomerID == uuid.Nil {
		log.Error("Missing customer_id in request body.")
		return simpleResponse(400), nil
	}

	res, err := h.contactHandler.StarSet(ctx, body.CustomerID, agentID, contactID)
	if err != nil {
		log.Errorf("Could not star the contact. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil //nolint:nilerr // failure is reported as a 500 response
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1ContactsIDStarsDelete handles DELETE /v1/contacts/{id}/stars/{agent_id}:
// the agent removes the star. Idempotent.
func (h *listenHandler) processV1ContactsIDStarsDelete(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1ContactsIDStarsDelete"})
	log.WithField("request", req).Debug("Received request.")

	contactID, agentID, ok := parseStarIDs(req.URI)
	if !ok {
		return simpleResponse(400), nil
	}

	var body request.V1DataContactsIDStars
	if len(req.Data) > 0 {
		_ = json.Unmarshal(req.Data, &body)
	}
	if body.CustomerID == uuid.Nil {
		log.Error("Missing customer_id in request body.")
		return simpleResponse(400), nil
	}

	if err := h.contactHandler.StarDelete(ctx, body.CustomerID, agentID, contactID); err != nil {
		log.Errorf("Could not remove the star. err: %v", err)
		return errorResponse(err), nil
	}

	return simpleResponse(200), nil
}

// parseStarsGetBody reads the customer and the agent of a star list request.
func parseStarsGetBody(data []byte) (request.V1DataStarsGet, bool) {
	var body request.V1DataStarsGet
	if len(data) > 0 {
		_ = json.Unmarshal(data, &body)
	}

	return body, body.CustomerID != uuid.Nil && body.AgentID != uuid.Nil
}

// processV1ContactsStarredGet handles GET /v1/contacts/starred?page_size&page_token: the
// contacts the agent has starred, in the contact list order (contact tm_create, newest first).
func (h *listenHandler) processV1ContactsStarredGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1ContactsStarredGet"})
	log.WithField("request", req).Debug("Received request.")

	size, token, ok := parseStarPaging(req.URI)
	if !ok {
		return simpleResponse(400), nil
	}

	body, ok := parseStarsGetBody(req.Data)
	if !ok {
		log.Error("Missing customer_id or agent_id in request body.")
		return simpleResponse(400), nil
	}

	res, err := h.contactHandler.ListStarred(ctx, body.CustomerID, body.AgentID, size, token)
	if err != nil {
		log.Errorf("Could not list the starred contacts. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil //nolint:nilerr // failure is reported as a 500 response
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1ContactStarsGet handles GET /v1/contact_stars?page_size&page_token: the agent's star
// rows, newest star first.
func (h *listenHandler) processV1ContactStarsGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1ContactStarsGet"})
	log.WithField("request", req).Debug("Received request.")

	size, token, ok := parseStarPaging(req.URI)
	if !ok {
		return simpleResponse(400), nil
	}

	body, ok := parseStarsGetBody(req.Data)
	if !ok {
		log.Error("Missing customer_id or agent_id in request body.")
		return simpleResponse(400), nil
	}

	res, err := h.contactHandler.StarList(ctx, body.CustomerID, body.AgentID, size, token)
	if err != nil {
		log.Errorf("Could not list the stars. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil //nolint:nilerr // failure is reported as a 500 response
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}
