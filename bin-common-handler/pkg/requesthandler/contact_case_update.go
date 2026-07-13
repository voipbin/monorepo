package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"

	cmkase "monorepo/bin-contact-manager/models/kase"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"
)

// ContactV1CaseUpdateContact attaches or detaches a case's contact via
// a direct contact_id write (VOIP-1253). contactID == uuid.Nil clears
// the attribution.
func (r *requestHandler) ContactV1CaseUpdateContact(ctx context.Context, customerID, caseID, contactID uuid.UUID) (*cmkase.Case, error) {
	uri := fmt.Sprintf("/v1/cases/%s", caseID)

	data := &cmrequest.V1DataCasesIDPut{CustomerID: customerID, ContactID: contactID}
	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPut, "contact/cases/<id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmkase.Case
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}
	return &res, nil
}
