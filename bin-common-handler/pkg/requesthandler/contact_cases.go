package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"

	cmcasenote "monorepo/bin-contact-manager/models/casenote"
	cmkase "monorepo/bin-contact-manager/models/kase"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"
)

// ContactV1CaseList lists cases from contact-manager, optionally filtered
// by status, owner, and/or contact_id (query-string filters; size/token
// are accepted for API symmetry with other list clients, but the
// v1_cases.go listenhandler does not currently implement pagination
// on this route, so nextToken is always returned empty).
func (r *requestHandler) ContactV1CaseList(
	ctx context.Context,
	customerID uuid.UUID,
	status, ownerType string,
	ownerID uuid.UUID,
	contactID uuid.UUID,
	size uint64,
	token string,
) ([]*cmkase.Case, string, error) {
	u := url.Values{}
	if status != "" {
		u.Set("status", status)
	}
	if ownerType != "" {
		u.Set("owner_type", ownerType)
	}
	if ownerID != uuid.Nil {
		u.Set("owner_id", ownerID.String())
	}
	if contactID != uuid.Nil {
		u.Set("contact_id", contactID.String())
	}

	uri := "/v1/cases?" + u.Encode()

	m, err := json.Marshal(cmrequest.V1DataCasesGet{CustomerID: customerID})
	if err != nil {
		return nil, "", err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/cases", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, "", err
	}

	var res []*cmkase.Case
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, "", errParse
	}

	return res, "", nil
}

// ContactV1CaseListUnresolved lists unresolved cases (open, contact_id
// IS NULL) from contact-manager. size/token are accepted for API
// symmetry with other list clients, but the v1_cases.go listenhandler
// does not currently implement pagination on this route, so nextToken
// is always returned empty.
func (r *requestHandler) ContactV1CaseListUnresolved(
	ctx context.Context,
	customerID uuid.UUID,
	size uint64,
	token string,
) ([]*cmkase.Case, string, error) {
	uri := "/v1/cases/unresolved"

	m, err := json.Marshal(cmrequest.V1DataCasesUnresolvedGet{CustomerID: customerID})
	if err != nil {
		return nil, "", err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/cases/unresolved", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, "", err
	}

	var res []*cmkase.Case
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, "", errParse
	}

	return res, "", nil
}

// ContactV1CaseGet gets a single case from contact-manager.
func (r *requestHandler) ContactV1CaseGet(ctx context.Context, customerID, id uuid.UUID) (*cmkase.Case, error) {
	uri := fmt.Sprintf("/v1/cases/%s", id)

	m, err := json.Marshal(cmrequest.V1DataCasesIDGet{CustomerID: customerID})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/cases/<id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmkase.Case
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1CaseClose closes a case in contact-manager. Note: the
// listenhandler's underlying casehandler.Close returns a CloseResult
// wrapper (Case + ClosedReason/ClosedByType/ClosedByID/AlreadyClosed),
// but processV1CasesIDClosePost marshals that CloseResult directly as
// the wire response body -- it does not unwrap to a bare Case. Since
// CloseResult embeds *kase.Case as a named field (not embedded), the
// wire shape is {"Case": {...}, "ClosedReason": ..., ...}, not a bare
// Case object. Decode into a local mirror struct and return .Case.
func (r *requestHandler) ContactV1CaseClose(
	ctx context.Context,
	customerID, id uuid.UUID,
	closedByType string,
	closedByID uuid.UUID,
) (*cmkase.Case, error) {
	uri := fmt.Sprintf("/v1/cases/%s/close", id)

	data := &cmrequest.V1DataCasesIDClose{
		CustomerID:   customerID,
		ClosedByType: closedByType,
		ClosedByID:   closedByID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/cases/<id>/close", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res struct {
		Case *cmkase.Case
	}
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res.Case, nil
}

// ContactV1CaseContinue creates a new, open case that continues a
// previously closed case in contact-manager.
func (r *requestHandler) ContactV1CaseContinue(
	ctx context.Context,
	customerID, id uuid.UUID,
	callerType string,
	callerID uuid.UUID,
	callerIsAdmin bool,
) (*cmkase.Case, error) {
	uri := fmt.Sprintf("/v1/cases/%s/continue", id)

	data := &cmrequest.V1DataCasesIDContinue{
		CustomerID:    customerID,
		CallerType:    callerType,
		CallerID:      callerID,
		CallerIsAdmin: callerIsAdmin,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/cases/<id>/continue", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmkase.Case
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1CaseNoteList lists notes for a case from contact-manager.
func (r *requestHandler) ContactV1CaseNoteList(ctx context.Context, customerID, caseID uuid.UUID) ([]*cmcasenote.CaseNote, error) {
	uri := fmt.Sprintf("/v1/cases/%s/notes", caseID)

	m, err := json.Marshal(cmrequest.V1DataCasesIDNotesGet{CustomerID: customerID})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/cases/<id>/notes", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []*cmcasenote.CaseNote
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// ContactV1CaseNoteCreate creates a note on a case in contact-manager.
func (r *requestHandler) ContactV1CaseNoteCreate(
	ctx context.Context,
	customerID, caseID uuid.UUID,
	authorType string,
	authorID *uuid.UUID,
	text string,
) (*cmcasenote.CaseNote, error) {
	uri := fmt.Sprintf("/v1/cases/%s/notes", caseID)

	data := &cmrequest.V1DataCasesIDNotesPost{
		CustomerID: customerID,
		AuthorType: authorType,
		AuthorID:   authorID,
		Text:       text,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/cases/<id>/notes", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcasenote.CaseNote
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1CaseNoteDelete deletes a case note in contact-manager.
func (r *requestHandler) ContactV1CaseNoteDelete(ctx context.Context, customerID, caseID, noteID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/cases/%s/notes/%s", caseID, noteID)

	m, err := json.Marshal(cmrequest.V1DataCasesIDNotesIDDelete{CustomerID: customerID})
	if err != nil {
		return err
	}

	_, err = r.sendRequestContact(ctx, uri, sock.RequestMethodDelete, "contact/cases/<id>/notes/<note-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	return err
}

// ContactV1CaseTagList lists tag ids assigned to a case in contact-manager.
func (r *requestHandler) ContactV1CaseTagList(ctx context.Context, customerID, caseID uuid.UUID) ([]uuid.UUID, error) {
	uri := fmt.Sprintf("/v1/cases/%s/tags", caseID)

	m, err := json.Marshal(cmrequest.V1DataCasesIDTagsGet{CustomerID: customerID})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/cases/<id>/tags", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []uuid.UUID
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// ContactV1CaseTagAdd assigns a tag to a case in contact-manager.
func (r *requestHandler) ContactV1CaseTagAdd(ctx context.Context, customerID, caseID, tagID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/cases/%s/tags", caseID)

	data := &cmrequest.V1DataCasesIDTagsPost{
		CustomerID: customerID,
		TagID:      tagID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/cases/<id>/tags", requestTimeoutDefault, 0, ContentTypeJSON, m)
	return err
}

// ContactV1CaseTagRemove unassigns a tag from a case in contact-manager.
func (r *requestHandler) ContactV1CaseTagRemove(ctx context.Context, customerID, caseID, tagID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/cases/%s/tags/%s", caseID, tagID)

	m, err := json.Marshal(cmrequest.V1DataCasesIDTagsIDDelete{CustomerID: customerID})
	if err != nil {
		return err
	}

	_, err = r.sendRequestContact(ctx, uri, sock.RequestMethodDelete, "contact/cases/<id>/tags/<tag-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	return err
}
