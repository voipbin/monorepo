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

// processV1ContactsIDCaseNotesGet handles GET /v1/contacts/{id}/case_notes
// request: the notes of every case currently attributed to the contact,
// newest first, with page_size/page_token pagination.
func (h *listenHandler) processV1ContactsIDCaseNotesGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1ContactsIDCaseNotesGet"})
	log.WithField("request", req).Debug("Received request.")

	u, err := url.Parse(req.URI)
	if err != nil {
		return simpleResponse(400), nil //nolint:nilerr // malformed request is reported as a 400 response
	}

	// /v1/contacts/<uuid>/case_notes
	parts := strings.Split(u.Path, "/")
	if len(parts) < 5 {
		return simpleResponse(400), nil
	}
	contactID := uuid.FromStringOrNil(parts[3])
	if contactID == uuid.Nil {
		return simpleResponse(400), nil
	}

	q := u.Query()
	tmpSize, _ := strconv.Atoi(q.Get(PageSize))
	pageSize := uint64(0)
	if tmpSize > 0 {
		pageSize = uint64(tmpSize)
	}
	pageToken := q.Get(PageToken)

	var body request.V1DataContactsIDCaseNotesGet
	if len(req.Data) > 0 {
		_ = json.Unmarshal(req.Data, &body)
	}
	if body.CustomerID == uuid.Nil {
		log.Error("Missing customer_id in request body.")
		return simpleResponse(400), nil
	}

	res, _, err := h.caseHandler.CaseNoteListByContact(ctx, body.CustomerID, contactID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not list case notes of the contact. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil //nolint:nilerr // failure is reported as a 500 response
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}
