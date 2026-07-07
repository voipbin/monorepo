package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-contact-manager/pkg/listenhandler/models/request"
)

// caseIDFromURI extracts the case id from a URI of the form
// /v1/cases/<uuid>... (parts[3]). Callers must have already matched a
// regex that guarantees the id segment is present and well-formed.
func caseIDFromURI(uri string) uuid.UUID {
	parts := strings.Split(uri, "/")
	if len(parts) < 4 {
		return uuid.Nil
	}
	return uuid.FromStringOrNil(parts[3])
}

// caseSubIDFromURI extracts a trailing sub-resource id from a URI of
// the form /v1/cases/<uuid>/<sub>/<uuid> (parts[5]).
func caseSubIDFromURI(uri string) uuid.UUID {
	parts := strings.Split(uri, "/")
	if len(parts) < 6 {
		return uuid.Nil
	}
	return uuid.FromStringOrNil(parts[5])
}

// processV1CasesGet handles GET /v1/cases?... request. Supports
// optional status and owner_type/owner_id filters (design §9).
func (h *listenHandler) processV1CasesGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1CasesGet"})
	log.WithField("request", req).Debug("Received request.")

	u, err := url.Parse(req.URI)
	if err != nil {
		return simpleResponse(400), nil
	}
	q := u.Query()

	status := q.Get("status")
	ownerType := commonidentity.OwnerType(q.Get("owner_type"))
	var ownerID uuid.UUID
	if s := q.Get("owner_id"); s != "" {
		ownerID = uuid.FromStringOrNil(s)
	}

	var body request.V1DataCasesGet
	if len(req.Data) > 0 {
		_ = json.Unmarshal(req.Data, &body)
	}
	if body.CustomerID == uuid.Nil {
		log.Error("Missing customer_id in request body.")
		return simpleResponse(400), nil
	}

	res, err := h.caseHandler.CaseList(ctx, body.CustomerID, status, ownerType, ownerID)
	if err != nil {
		log.Errorf("Could not list cases. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1CasesUnresolvedGet handles GET /v1/cases/unresolved request.
func (h *listenHandler) processV1CasesUnresolvedGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1CasesUnresolvedGet"})
	log.WithField("request", req).Debug("Received request.")

	var body request.V1DataCasesUnresolvedGet
	if len(req.Data) > 0 {
		_ = json.Unmarshal(req.Data, &body)
	}
	if body.CustomerID == uuid.Nil {
		log.Error("Missing customer_id in request body.")
		return simpleResponse(400), nil
	}

	res, err := h.caseHandler.CaseListUnresolved(ctx, body.CustomerID)
	if err != nil {
		log.Errorf("Could not list unresolved cases. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1CasesIDGet handles GET /v1/cases/{id} request.
func (h *listenHandler) processV1CasesIDGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1CasesIDGet"})
	log.WithField("request", req).Debug("Received request.")

	id := caseIDFromURI(req.URI)
	if id == uuid.Nil {
		return simpleResponse(400), nil
	}

	var body request.V1DataCasesIDGet
	if len(req.Data) > 0 {
		_ = json.Unmarshal(req.Data, &body)
	}
	if body.CustomerID == uuid.Nil {
		log.Error("Missing customer_id in request body.")
		return simpleResponse(400), nil
	}

	res, err := h.caseHandler.CaseGet(ctx, body.CustomerID, id)
	if err != nil {
		log.Errorf("Could not get case. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1CasesIDClosePost handles POST /v1/cases/{id}/close request.
func (h *listenHandler) processV1CasesIDClosePost(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1CasesIDClosePost"})
	log.WithField("request", req).Debug("Received request.")

	id := caseIDFromURI(req.URI)
	if id == uuid.Nil {
		return simpleResponse(400), nil
	}

	var body request.V1DataCasesIDClose
	if err := json.Unmarshal(req.Data, &body); err != nil {
		log.Errorf("Could not unmarshal request body. err: %v", err)
		return simpleResponse(400), nil
	}
	if body.CustomerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	res, err := h.caseHandler.Close(ctx, body.CustomerID, id, commonidentity.OwnerType(body.ClosedByType), body.ClosedByID)
	if err != nil {
		log.Errorf("Could not close case. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1CasesIDContinuePost handles POST /v1/cases/{id}/continue
// request.
func (h *listenHandler) processV1CasesIDContinuePost(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1CasesIDContinuePost"})
	log.WithField("request", req).Debug("Received request.")

	id := caseIDFromURI(req.URI)
	if id == uuid.Nil {
		return simpleResponse(400), nil
	}

	var body request.V1DataCasesIDContinue
	if err := json.Unmarshal(req.Data, &body); err != nil {
		log.Errorf("Could not unmarshal request body. err: %v", err)
		return simpleResponse(400), nil
	}
	if body.CustomerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	res, err := h.caseHandler.Continue(ctx, body.CustomerID, id, commonidentity.OwnerType(body.CallerType), body.CallerID, body.CallerIsAdmin)
	if err != nil {
		log.Errorf("Could not continue case. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1CasesIDNotesGet handles GET /v1/cases/{id}/notes request.
func (h *listenHandler) processV1CasesIDNotesGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1CasesIDNotesGet"})
	log.WithField("request", req).Debug("Received request.")

	id := caseIDFromURI(req.URI)
	if id == uuid.Nil {
		return simpleResponse(400), nil
	}

	var body request.V1DataCasesIDNotesGet
	if len(req.Data) > 0 {
		_ = json.Unmarshal(req.Data, &body)
	}
	if body.CustomerID == uuid.Nil {
		log.Error("Missing customer_id in request body.")
		return simpleResponse(400), nil
	}

	res, err := h.caseHandler.CaseNoteListByCase(ctx, body.CustomerID, id)
	if err != nil {
		log.Errorf("Could not list case notes. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1CasesIDNotesPost handles POST /v1/cases/{id}/notes request.
func (h *listenHandler) processV1CasesIDNotesPost(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1CasesIDNotesPost"})
	log.WithField("request", req).Debug("Received request.")

	id := caseIDFromURI(req.URI)
	if id == uuid.Nil {
		return simpleResponse(400), nil
	}

	var body request.V1DataCasesIDNotesPost
	if err := json.Unmarshal(req.Data, &body); err != nil {
		log.Errorf("Could not unmarshal request body. err: %v", err)
		return simpleResponse(400), nil
	}
	if body.CustomerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	res, err := h.caseHandler.CaseNoteCreate(ctx, body.CustomerID, id, body.AuthorType, body.AuthorID, body.Text)
	if err != nil {
		log.Errorf("Could not create case note. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1CasesIDNotesIDDelete handles
// DELETE /v1/cases/{id}/notes/{note_id} request.
func (h *listenHandler) processV1CasesIDNotesIDDelete(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1CasesIDNotesIDDelete"})
	log.WithField("request", req).Debug("Received request.")

	id := caseIDFromURI(req.URI)
	if id == uuid.Nil {
		return simpleResponse(400), nil
	}
	noteID := caseSubIDFromURI(req.URI)
	if noteID == uuid.Nil {
		return simpleResponse(400), nil
	}

	var body request.V1DataCasesIDNotesIDDelete
	if err := json.Unmarshal(req.Data, &body); err != nil {
		log.Errorf("Could not unmarshal request body. err: %v", err)
		return simpleResponse(400), nil
	}
	if body.CustomerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	if err := h.caseHandler.CaseNoteDelete(ctx, body.CustomerID, id, noteID); err != nil {
		log.Errorf("Could not delete case note. err: %v", err)
		return errorResponse(err), nil
	}

	return simpleResponse(200), nil
}

// processV1CasesIDTagsGet handles GET /v1/cases/{id}/tags request.
func (h *listenHandler) processV1CasesIDTagsGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1CasesIDTagsGet"})
	log.WithField("request", req).Debug("Received request.")

	id := caseIDFromURI(req.URI)
	if id == uuid.Nil {
		return simpleResponse(400), nil
	}

	var body request.V1DataCasesIDTagsGet
	if len(req.Data) > 0 {
		_ = json.Unmarshal(req.Data, &body)
	}
	if body.CustomerID == uuid.Nil {
		log.Error("Missing customer_id in request body.")
		return simpleResponse(400), nil
	}

	res, err := h.caseHandler.CaseTagList(ctx, body.CustomerID, id)
	if err != nil {
		log.Errorf("Could not list case tags. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1CasesIDTagsPost handles POST /v1/cases/{id}/tags request.
func (h *listenHandler) processV1CasesIDTagsPost(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1CasesIDTagsPost"})
	log.WithField("request", req).Debug("Received request.")

	id := caseIDFromURI(req.URI)
	if id == uuid.Nil {
		return simpleResponse(400), nil
	}

	var body request.V1DataCasesIDTagsPost
	if err := json.Unmarshal(req.Data, &body); err != nil {
		log.Errorf("Could not unmarshal request body. err: %v", err)
		return simpleResponse(400), nil
	}
	if body.CustomerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	if err := h.caseHandler.CaseTagAdd(ctx, body.CustomerID, id, body.TagID); err != nil {
		log.Errorf("Could not add case tag. err: %v", err)
		return errorResponse(err), nil
	}

	return simpleResponse(200), nil
}

// processV1CasesIDTagsIDDelete handles
// DELETE /v1/cases/{id}/tags/{tag_id} request.
func (h *listenHandler) processV1CasesIDTagsIDDelete(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1CasesIDTagsIDDelete"})
	log.WithField("request", req).Debug("Received request.")

	id := caseIDFromURI(req.URI)
	if id == uuid.Nil {
		return simpleResponse(400), nil
	}
	tagID := caseSubIDFromURI(req.URI)
	if tagID == uuid.Nil {
		return simpleResponse(400), nil
	}

	var body request.V1DataCasesIDTagsIDDelete
	if err := json.Unmarshal(req.Data, &body); err != nil {
		log.Errorf("Could not unmarshal request body. err: %v", err)
		return simpleResponse(400), nil
	}
	if body.CustomerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	if err := h.caseHandler.CaseTagRemove(ctx, body.CustomerID, id, tagID); err != nil {
		log.Errorf("Could not remove case tag. err: %v", err)
		return errorResponse(err), nil
	}

	return simpleResponse(200), nil
}
