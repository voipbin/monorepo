package listenhandler

import (
	"context"
	"encoding/json"
	"testing"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_ProcessV1CasesIDPut_Success verifies PUT /v1/cases/{id} with a
// valid UUID contact_id in the body succeeds and returns 200 with the
// updated case, delegating to caseHandler.UpdateContact with the
// parsed customerID/caseID/contactID.
func Test_ProcessV1CasesIDPut_Success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockCase := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-0009-0009-0009-000000000001")
	caseID := uuid.FromStringOrNil("aaaaaaaa-0009-0009-0009-000000000002")
	contactID := uuid.FromStringOrNil("aaaaaaaa-0009-0009-0009-000000000003")

	body, _ := json.Marshal(map[string]any{
		"customer_id": customerID.String(),
		"contact_id":  contactID.String(),
	})
	req := &sock.Request{URI: "/v1/cases/" + caseID.String(), Method: sock.RequestMethodPut, Data: body}

	mockCase.EXPECT().UpdateContact(ctx, customerID, caseID, contactID).
		Return(&kase.Case{ID: caseID, CustomerID: customerID, ContactID: &contactID}, nil)

	res, err := h.processV1CasesIDPut(ctx, req)
	if err != nil {
		t.Fatalf("processV1CasesIDPut() error = %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("StatusCode = %v, want 200", res.StatusCode)
	}

	var got kase.Case
	if err := json.Unmarshal(res.Data, &got); err != nil {
		t.Fatalf("could not unmarshal response: %v", err)
	}
	if got.ID != caseID || got.ContactID == nil || *got.ContactID != contactID {
		t.Errorf("unexpected response case: %+v", got)
	}
}

// Test_ProcessV1CasesIDPut_Detach verifies a PUT with contact_id
// omitted (uuid.Nil) is treated as a detach and passed through as
// uuid.Nil to UpdateContact.
func Test_ProcessV1CasesIDPut_Detach(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockCase := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-0009-0009-0009-000000000004")
	caseID := uuid.FromStringOrNil("aaaaaaaa-0009-0009-0009-000000000005")

	body, _ := json.Marshal(map[string]any{"customer_id": customerID.String()})
	req := &sock.Request{URI: "/v1/cases/" + caseID.String(), Method: sock.RequestMethodPut, Data: body}

	mockCase.EXPECT().UpdateContact(ctx, customerID, caseID, uuid.Nil).
		Return(&kase.Case{ID: caseID, CustomerID: customerID}, nil)

	res, err := h.processV1CasesIDPut(ctx, req)
	if err != nil {
		t.Fatalf("processV1CasesIDPut() error = %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("StatusCode = %v, want 200", res.StatusCode)
	}
}

// Test_ProcessV1CasesIDPut_InvalidCaseID verifies a missing/invalid
// case ID in the URI returns 400 without calling the handler.
func Test_ProcessV1CasesIDPut_InvalidCaseID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, _ := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	req := &sock.Request{URI: "/v1/cases/", Method: sock.RequestMethodPut, Data: []byte(`{}`)}

	res, err := h.processV1CasesIDPut(ctx, req)
	if err != nil {
		t.Fatalf("processV1CasesIDPut() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("StatusCode = %v, want 400", res.StatusCode)
	}
}

// Test_ProcessV1CasesIDPut_MalformedBody verifies a malformed JSON
// body returns 400 without calling the handler.
func Test_ProcessV1CasesIDPut_MalformedBody(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, _ := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	caseID := uuid.FromStringOrNil("aaaaaaaa-0009-0009-0009-000000000006")
	req := &sock.Request{URI: "/v1/cases/" + caseID.String(), Method: sock.RequestMethodPut, Data: []byte(`{not-json`)}

	res, err := h.processV1CasesIDPut(ctx, req)
	if err != nil {
		t.Fatalf("processV1CasesIDPut() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("StatusCode = %v, want 400", res.StatusCode)
	}
}

// Test_ProcessV1CasesIDPut_MissingCustomerID verifies a missing
// customer_id in the body returns 400 without calling the handler.
func Test_ProcessV1CasesIDPut_MissingCustomerID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, _ := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	caseID := uuid.FromStringOrNil("aaaaaaaa-0009-0009-0009-000000000007")
	body, _ := json.Marshal(map[string]any{})
	req := &sock.Request{URI: "/v1/cases/" + caseID.String(), Method: sock.RequestMethodPut, Data: body}

	res, err := h.processV1CasesIDPut(ctx, req)
	if err != nil {
		t.Fatalf("processV1CasesIDPut() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("StatusCode = %v, want 400", res.StatusCode)
	}
}

// Test_ProcessV1CasesIDPut_ErrorPropagation verifies that an error
// returned by caseHandler.UpdateContact (e.g. cross-tenant
// dbhandler.ErrNotFound) is correctly propagated to the HTTP response
// via errorResponse, not swallowed or turned into a 200.
func Test_ProcessV1CasesIDPut_ErrorPropagation(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockCase := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-0009-0009-0009-000000000008")
	caseID := uuid.FromStringOrNil("aaaaaaaa-0009-0009-0009-000000000009")
	contactID := uuid.FromStringOrNil("aaaaaaaa-0009-0009-0009-00000000000a")

	body, _ := json.Marshal(map[string]any{
		"customer_id": customerID.String(),
		"contact_id":  contactID.String(),
	})
	req := &sock.Request{URI: "/v1/cases/" + caseID.String(), Method: sock.RequestMethodPut, Data: body}

	mockCase.EXPECT().UpdateContact(ctx, customerID, caseID, contactID).
		Return(nil, dbhandler.ErrNotFound)

	res, err := h.processV1CasesIDPut(ctx, req)
	if err != nil {
		t.Fatalf("processV1CasesIDPut() error = %v", err)
	}
	if res.StatusCode != 404 {
		t.Errorf("StatusCode = %v, want 404", res.StatusCode)
	}
}
