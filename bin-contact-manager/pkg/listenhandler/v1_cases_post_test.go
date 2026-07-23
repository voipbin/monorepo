package listenhandler

import (
	"context"
	"encoding/json"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/kase"
)

// Test_ProcessV1CasesPost_CreatesCase verifies POST /v1/cases
// unmarshals the request body and delegates to caseHandler.Create
// (design VOIP-1243 §4's listenhandler layer).
func Test_ProcessV1CasesPost_CreatesCase(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockCase := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-0101-0101-0101-000000000001")
	caseID := uuid.FromStringOrNil("aaaaaaaa-0101-0101-0101-000000000002")

	reqBody := map[string]any{
		"customer_id": customerID.String(),
		"self":        map[string]any{"type": "tel", "target": "+155****0101"},
		"peer":        map[string]any{"type": "tel", "target": "+155****9101"},
		"reference_type": "call",
		"name":   "VIP",
		"detail": "escalated",
	}
	body, _ := json.Marshal(reqBody)
	req := &sock.Request{
		URI:    "/v1/cases",
		Method: sock.RequestMethodPost,
		Data:   body,
	}

	mockCase.EXPECT().Create(
		ctx, customerID,
		commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0101"},
		commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****9101"}, "call", "VIP", "escalated", "",
	).Return(&kase.Case{ID: caseID, CustomerID: customerID}, nil)

	res, err := h.processV1CasesPost(ctx, req)
	if err != nil {
		t.Fatalf("processV1CasesPost() error = %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("StatusCode = %v, want 200", res.StatusCode)
	}
}

// Test_ProcessV1CasesPost_MissingCustomerID verifies a missing
// customer_id yields 400 without calling caseHandler.Create.
func Test_ProcessV1CasesPost_MissingCustomerID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, _ := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	body, _ := json.Marshal(map[string]any{})
	req := &sock.Request{URI: "/v1/cases", Method: sock.RequestMethodPost, Data: body}

	res, err := h.processV1CasesPost(ctx, req)
	if err != nil {
		t.Fatalf("processV1CasesPost() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("StatusCode = %v, want 400", res.StatusCode)
	}
}

// Test_RouteDispatch_PostCasesVsGetCasesQuery verifies routing-order
// correctness (design §4's note): a bare POST /v1/cases does not
// collide with the GET /v1/cases?... query-string route, and vice
// versa for GET.
func Test_RouteDispatch_PostCasesVsGetCasesQuery(t *testing.T) {
	if !regV1Cases.MatchString("/v1/cases") {
		t.Errorf("regV1Cases should match a bare /v1/cases URI")
	}
	if regV1Cases.MatchString("/v1/cases?status=open") {
		t.Errorf("regV1Cases should NOT match a query-string URI")
	}
	if !regV1CasesGet.MatchString("/v1/cases?status=open") {
		t.Errorf("regV1CasesGet should match a query-string URI")
	}
	if regV1CasesGet.MatchString("/v1/cases") {
		t.Errorf("regV1CasesGet should NOT match a bare /v1/cases URI")
	}
}
