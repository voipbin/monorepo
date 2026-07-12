package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-contact-manager/pkg/listenhandler/models/request"
)

// processV1CasesIDResolutionsPost handles POST /v1/cases/{id}/resolutions
// (VOIP-1252): attaches a Case to a Contact by creating a case-level
// Resolution, delegating to casehandler.ResolutionCreateCaseLevel.
func (h *listenHandler) processV1CasesIDResolutionsPost(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1CasesIDResolutionsPost"})
	log.WithField("request", req).Debug("Received request.")

	id := caseIDFromURI(req.URI)
	if id == uuid.Nil {
		return simpleResponse(400), nil
	}

	var body request.V1DataCasesIDResolutionsPost
	if err := json.Unmarshal(req.Data, &body); err != nil {
		log.Errorf("Could not unmarshal request body. err: %v", err)
		return simpleResponse(400), nil
	}
	if body.CustomerID == uuid.Nil || body.ContactID == uuid.Nil {
		return simpleResponse(400), nil
	}

	res, err := h.caseHandler.ResolutionCreateCaseLevel(
		ctx, body.CustomerID, id, body.ContactID,
		body.ResolutionType, body.ResolvedByType, body.ResolvedByID,
	)
	if err != nil {
		log.Errorf("Could not create case resolution. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1CasesIDResolutionsIDDelete handles
// DELETE /v1/cases/{id}/resolutions/{resolution_id} (VOIP-1252): undoes
// a case-level Contact attribution, delegating to
// casehandler.ResolutionDeleteCaseLevel.
func (h *listenHandler) processV1CasesIDResolutionsIDDelete(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1CasesIDResolutionsIDDelete"})
	log.WithField("request", req).Debug("Received request.")

	id := caseIDFromURI(req.URI)
	if id == uuid.Nil {
		return simpleResponse(400), nil
	}
	resolutionID := caseSubIDFromURI(req.URI)
	if resolutionID == uuid.Nil {
		return simpleResponse(400), nil
	}

	var body request.V1DataCasesIDResolutionsIDDelete
	if err := json.Unmarshal(req.Data, &body); err != nil {
		log.Errorf("Could not unmarshal request body. err: %v", err)
		return simpleResponse(400), nil
	}
	if body.CustomerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	if err := h.caseHandler.ResolutionDeleteCaseLevel(ctx, body.CustomerID, id, resolutionID); err != nil {
		log.Errorf("Could not delete case resolution. err: %v", err)
		return errorResponse(err), nil
	}

	return simpleResponse(200), nil
}
