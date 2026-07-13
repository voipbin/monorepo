package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/pkg/listenhandler/models/request"
)

// processV1CasesIDPut handles PUT /v1/cases/{id} (VOIP-1253): attaches
// or detaches a Case's Contact via a direct contact_id write.
func (h *listenHandler) processV1CasesIDPut(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	id := caseIDFromURI(req.URI)
	if id == uuid.Nil {
		return simpleResponse(400), nil
	}

	var body request.V1DataCasesIDPut
	if err := json.Unmarshal(req.Data, &body); err != nil {
		return simpleResponse(400), nil
	}
	if body.CustomerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	res, err := h.caseHandler.UpdateContact(ctx, body.CustomerID, id, body.ContactID)
	if err != nil {
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil
	}
	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}
