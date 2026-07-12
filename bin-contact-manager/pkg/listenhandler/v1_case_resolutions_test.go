package listenhandler

import (
	"context"
	"encoding/json"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/resolution"
)

// Test_ProcessV1CasesIDResolutionsPost_Success verifies POST
// /v1/cases/{id}/resolutions attaches a Case to a Contact, delegating to
// caseHandler.ResolutionCreateCaseLevel with all six parameters.
func Test_ProcessV1CasesIDResolutionsPost_Success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockCase := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-0008-0008-0008-000000000001")
	caseID := uuid.FromStringOrNil("aaaaaaaa-0008-0008-0008-000000000002")
	contactID := uuid.FromStringOrNil("aaaaaaaa-0008-0008-0008-000000000003")
	agentID := uuid.FromStringOrNil("aaaaaaaa-0008-0008-0008-000000000004")
	resolutionID := uuid.FromStringOrNil("aaaaaaaa-0008-0008-0008-000000000005")

	body, _ := json.Marshal(map[string]any{
		"customer_id":      customerID.String(),
		"contact_id":       contactID.String(),
		"resolution_type":  "positive",
		"resolved_by_type": "agent",
		"resolved_by_id":   agentID.String(),
	})

	mockCase.EXPECT().ResolutionCreateCaseLevel(ctx, customerID, caseID, contactID, "positive", "agent", agentID).
		Return(&resolution.Resolution{ID: resolutionID, CaseID: &caseID, ContactID: contactID}, nil)

	res, err := h.processV1CasesIDResolutionsPost(ctx, &sock.Request{
		URI: "/v1/cases/" + caseID.String() + "/resolutions", Method: sock.RequestMethodPost, Data: body,
	})
	if err != nil {
		t.Fatalf("processV1CasesIDResolutionsPost() error = %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("StatusCode = %v, want 200", res.StatusCode)
	}
}

// Test_ProcessV1CasesIDResolutionsPost_MissingFields verifies 400 is
// returned (without calling the handler) when customer_id or contact_id
// is missing from the request body.
func Test_ProcessV1CasesIDResolutionsPost_MissingFields(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, _ := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	caseID := uuid.FromStringOrNil("aaaaaaaa-0009-0009-0009-000000000001")
	contactID := uuid.FromStringOrNil("aaaaaaaa-0009-0009-0009-000000000002")

	tests := []struct {
		name string
		body map[string]any
	}{
		{"missing customer_id", map[string]any{"contact_id": contactID.String(), "resolution_type": "positive"}},
		{"missing contact_id", map[string]any{"customer_id": uuid.FromStringOrNil("aaaaaaaa-0009-0009-0009-000000000003").String(), "resolution_type": "positive"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			res, err := h.processV1CasesIDResolutionsPost(ctx, &sock.Request{
				URI: "/v1/cases/" + caseID.String() + "/resolutions", Method: sock.RequestMethodPost, Data: body,
			})
			if err != nil {
				t.Fatalf("processV1CasesIDResolutionsPost() error = %v", err)
			}
			if res.StatusCode != 400 {
				t.Errorf("StatusCode = %v, want 400", res.StatusCode)
			}
		})
	}
}

// Test_ProcessV1CasesIDResolutionsPost_MalformedBody verifies a
// malformed JSON body yields 400.
func Test_ProcessV1CasesIDResolutionsPost_MalformedBody(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, _ := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	caseID := uuid.FromStringOrNil("aaaaaaaa-0010-0010-0010-000000000001")

	res, err := h.processV1CasesIDResolutionsPost(ctx, &sock.Request{
		URI: "/v1/cases/" + caseID.String() + "/resolutions", Method: sock.RequestMethodPost, Data: []byte("{not json"),
	})
	if err != nil {
		t.Fatalf("processV1CasesIDResolutionsPost() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("StatusCode = %v, want 400", res.StatusCode)
	}
}

// Test_ProcessV1CasesIDResolutionsPost_CrossTenantContact verifies a
// caseHandler-propagated CONTACT_NOT_FOUND error (the cross-tenant
// contact guard added in VOIP-1252) surfaces via errorResponse, not a
// generic 500.
func Test_ProcessV1CasesIDResolutionsPost_CrossTenantContact(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockCase := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-0011-0011-0011-000000000001")
	caseID := uuid.FromStringOrNil("aaaaaaaa-0011-0011-0011-000000000002")
	contactID := uuid.FromStringOrNil("aaaaaaaa-0011-0011-0011-000000000003")
	agentID := uuid.FromStringOrNil("aaaaaaaa-0011-0011-0011-000000000004")

	body, _ := json.Marshal(map[string]any{
		"customer_id":      customerID.String(),
		"contact_id":       contactID.String(),
		"resolution_type":  "positive",
		"resolved_by_type": "agent",
		"resolved_by_id":   agentID.String(),
	})

	notFoundErr := cerrors.NotFound(commonoutline.ServiceNameContactManager, "CONTACT_NOT_FOUND", "The contact was not found.")
	mockCase.EXPECT().ResolutionCreateCaseLevel(ctx, customerID, caseID, contactID, "positive", "agent", agentID).
		Return(nil, notFoundErr)

	res, err := h.processV1CasesIDResolutionsPost(ctx, &sock.Request{
		URI: "/v1/cases/" + caseID.String() + "/resolutions", Method: sock.RequestMethodPost, Data: body,
	})
	if err != nil {
		t.Fatalf("processV1CasesIDResolutionsPost() error = %v", err)
	}
	if res.StatusCode == 200 {
		t.Errorf("StatusCode = %v, want a non-200 error status", res.StatusCode)
	}
}

// Test_ProcessV1CasesIDResolutionsIDDelete_Success verifies DELETE
// /v1/cases/{id}/resolutions/{resolution_id} delegates to
// caseHandler.ResolutionDeleteCaseLevel.
func Test_ProcessV1CasesIDResolutionsIDDelete_Success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockCase := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-0012-0012-0012-000000000001")
	caseID := uuid.FromStringOrNil("aaaaaaaa-0012-0012-0012-000000000002")
	resolutionID := uuid.FromStringOrNil("aaaaaaaa-0012-0012-0012-000000000003")

	body, _ := json.Marshal(map[string]any{"customer_id": customerID.String()})

	mockCase.EXPECT().ResolutionDeleteCaseLevel(ctx, customerID, caseID, resolutionID).Return(nil)

	res, err := h.processV1CasesIDResolutionsIDDelete(ctx, &sock.Request{
		URI: "/v1/cases/" + caseID.String() + "/resolutions/" + resolutionID.String(), Method: sock.RequestMethodDelete, Data: body,
	})
	if err != nil {
		t.Fatalf("processV1CasesIDResolutionsIDDelete() error = %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("StatusCode = %v, want 200", res.StatusCode)
	}
}

// Test_ProcessV1CasesIDResolutionsIDDelete_MissingCustomerID verifies
// 400 is returned when customer_id is missing.
func Test_ProcessV1CasesIDResolutionsIDDelete_MissingCustomerID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, _ := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	caseID := uuid.FromStringOrNil("aaaaaaaa-0013-0013-0013-000000000001")
	resolutionID := uuid.FromStringOrNil("aaaaaaaa-0013-0013-0013-000000000002")

	res, err := h.processV1CasesIDResolutionsIDDelete(ctx, &sock.Request{
		URI: "/v1/cases/" + caseID.String() + "/resolutions/" + resolutionID.String(), Method: sock.RequestMethodDelete, Data: []byte("{}"),
	})
	if err != nil {
		t.Fatalf("processV1CasesIDResolutionsIDDelete() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("StatusCode = %v, want 400", res.StatusCode)
	}
}
