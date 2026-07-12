package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"

	cmresolution "monorepo/bin-contact-manager/models/resolution"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"
)

// ContactV1CaseResolutionCreate attaches a case to a contact by creating
// a case-level Resolution in contact-manager (VOIP-1252).
func (r *requestHandler) ContactV1CaseResolutionCreate(
	ctx context.Context,
	customerID, caseID, contactID uuid.UUID,
	resolutionType, resolvedByType string,
	resolvedByID uuid.UUID,
) (*cmresolution.Resolution, error) {
	uri := fmt.Sprintf("/v1/cases/%s/resolutions", caseID)

	data := &cmrequest.V1DataCasesIDResolutionsPost{
		CustomerID:     customerID,
		ContactID:      contactID,
		ResolutionType: resolutionType,
		ResolvedByType: resolvedByType,
		ResolvedByID:   resolvedByID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/cases/<id>/resolutions", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmresolution.Resolution
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1CaseResolutionDelete undoes a case-level Contact attribution
// (VOIP-1252) by soft-deleting the Resolution row in contact-manager.
func (r *requestHandler) ContactV1CaseResolutionDelete(ctx context.Context, customerID, caseID, resolutionID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/cases/%s/resolutions/%s", caseID, resolutionID)

	data := &cmrequest.V1DataCasesIDResolutionsIDDelete{CustomerID: customerID}

	m, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = r.sendRequestContact(ctx, uri, sock.RequestMethodDelete, "contact/cases/<id>/resolutions/<resolution-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	return err
}
