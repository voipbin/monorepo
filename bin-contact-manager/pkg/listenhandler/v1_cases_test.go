package listenhandler

import (
	"context"
	"encoding/json"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/casenote"
	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/addresshandler"
	"monorepo/bin-contact-manager/pkg/casehandler"
	"monorepo/bin-contact-manager/pkg/contacthandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

func newTestListenHandlerWithCase(mc *gomock.Controller) (*listenHandler, *casehandler.MockCaseHandler) {
	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)
	mockAddr := addresshandler.NewMockAddressHandler(mc)
	mockCase := casehandler.NewMockCaseHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
		addressHandler: mockAddr,
		caseHandler:    mockCase,
	}
	return h, mockCase
}

// Test_ProcessV1CasesGet_ListsWithFilters verifies GET /v1/cases?...
// parses status/owner_type/owner_id query filters and reads customer_id
// from the request body, delegating to caseHandler.CaseList.
func Test_ProcessV1CasesGet_ListsWithFilters(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockCase := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-0001-0001-0001-000000000001")
	ownerID := uuid.FromStringOrNil("aaaaaaaa-0001-0001-0001-000000000002")
	caseID := uuid.FromStringOrNil("aaaaaaaa-0001-0001-0001-000000000003")

	body, _ := json.Marshal(map[string]any{"customer_id": customerID.String()})
	req := &sock.Request{
		URI:    "/v1/cases?status=open&owner_type=agent&owner_id=" + ownerID.String(),
		Method: sock.RequestMethodGet,
		Data:   body,
	}

	mockCase.EXPECT().CaseList(ctx, customerID, "open", commonidentity.OwnerTypeAgent, ownerID).
		Return([]*kase.Case{{ID: caseID}}, nil)

	res, err := h.processV1CasesGet(ctx, req)
	if err != nil {
		t.Fatalf("processV1CasesGet() error = %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("StatusCode = %v, want 200", res.StatusCode)
	}
}

// Test_ProcessV1CasesGet_MissingCustomerID verifies a missing
// customer_id yields 400, without calling the handler.
func Test_ProcessV1CasesGet_MissingCustomerID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, _ := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	req := &sock.Request{URI: "/v1/cases?status=open", Method: sock.RequestMethodGet}

	res, err := h.processV1CasesGet(ctx, req)
	if err != nil {
		t.Fatalf("processV1CasesGet() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("StatusCode = %v, want 400", res.StatusCode)
	}
}

// Test_ProcessV1CasesUnresolvedGet verifies GET /v1/cases/unresolved.
func Test_ProcessV1CasesUnresolvedGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockCase := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-0002-0002-0002-000000000001")
	body, _ := json.Marshal(map[string]any{"customer_id": customerID.String()})
	req := &sock.Request{URI: "/v1/cases/unresolved", Method: sock.RequestMethodGet, Data: body}

	mockCase.EXPECT().CaseListUnresolved(ctx, customerID).Return([]*kase.Case{}, nil)

	res, err := h.processV1CasesUnresolvedGet(ctx, req)
	if err != nil {
		t.Fatalf("processV1CasesUnresolvedGet() error = %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("StatusCode = %v, want 200", res.StatusCode)
	}
}

// Test_ProcessV1CasesIDGet_CrossTenantMapsToNotFound verifies that a
// caseHandler.CaseGet cross-tenant rejection (dbhandler.ErrNotFound)
// maps to a 404 response at the listenhandler layer — the RPC surface
// must not leak existence.
func Test_ProcessV1CasesIDGet_CrossTenantMapsToNotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockCase := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-0003-0003-0003-000000000001")
	caseID := uuid.FromStringOrNil("aaaaaaaa-0003-0003-0003-000000000002")
	body, _ := json.Marshal(map[string]any{"customer_id": customerID.String()})
	req := &sock.Request{URI: "/v1/cases/" + caseID.String(), Method: sock.RequestMethodGet, Data: body}

	mockCase.EXPECT().CaseGet(ctx, customerID, caseID).Return(nil, dbhandler.ErrNotFound)

	res, err := h.processV1CasesIDGet(ctx, req)
	if err != nil {
		t.Fatalf("processV1CasesIDGet() error = %v", err)
	}
	if res.StatusCode != 404 {
		t.Errorf("StatusCode = %v, want 404", res.StatusCode)
	}
}

// Test_ProcessV1CasesIDClosePost verifies POST /v1/cases/{id}/close.
func Test_ProcessV1CasesIDClosePost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockCase := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-0004-0004-0004-000000000001")
	caseID := uuid.FromStringOrNil("aaaaaaaa-0004-0004-0004-000000000002")
	closedByID := uuid.FromStringOrNil("aaaaaaaa-0004-0004-0004-000000000003")

	body, _ := json.Marshal(map[string]any{
		"customer_id":    customerID.String(),
		"closed_by_type": "agent",
		"closed_by_id":   closedByID.String(),
	})
	req := &sock.Request{URI: "/v1/cases/" + caseID.String() + "/close", Method: sock.RequestMethodPost, Data: body}

	mockCase.EXPECT().Close(ctx, customerID, caseID, commonidentity.OwnerTypeAgent, closedByID).
		Return(&casehandler.CloseResult{Case: &kase.Case{ID: caseID}}, nil)

	res, err := h.processV1CasesIDClosePost(ctx, req)
	if err != nil {
		t.Fatalf("processV1CasesIDClosePost() error = %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("StatusCode = %v, want 200", res.StatusCode)
	}
}

// Test_ProcessV1CasesIDContinuePost verifies
// POST /v1/cases/{id}/continue.
func Test_ProcessV1CasesIDContinuePost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockCase := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-0005-0005-0005-000000000001")
	caseID := uuid.FromStringOrNil("aaaaaaaa-0005-0005-0005-000000000002")
	callerID := uuid.FromStringOrNil("aaaaaaaa-0005-0005-0005-000000000003")

	body, _ := json.Marshal(map[string]any{
		"customer_id":     customerID.String(),
		"caller_type":     "agent",
		"caller_id":       callerID.String(),
		"caller_is_admin": false,
	})
	req := &sock.Request{URI: "/v1/cases/" + caseID.String() + "/continue", Method: sock.RequestMethodPost, Data: body}

	mockCase.EXPECT().Continue(ctx, customerID, caseID, commonidentity.OwnerTypeAgent, callerID, false).
		Return(&kase.Case{ID: caseID}, nil)

	res, err := h.processV1CasesIDContinuePost(ctx, req)
	if err != nil {
		t.Fatalf("processV1CasesIDContinuePost() error = %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("StatusCode = %v, want 200", res.StatusCode)
	}
}

// Test_ProcessV1CasesIDContinuePost_ErrorMapping is a regression test
// (round-1 Phase 5 review defect): casehandler.Continue's two
// domain-specific sentinels (ErrCaseNotClosed, ErrCaseContinueForbidden)
// must be typed *cerrors.VoipbinError so errorResponse() maps them to
// the OpenAPI spec's declared 400/403, not a generic 500.
func Test_ProcessV1CasesIDContinuePost_ErrorMapping(t *testing.T) {
	tests := []struct {
		name         string
		continueErr  error
		expectStatus int
	}{
		{
			name:         "source case not closed maps to 400",
			continueErr:  casehandler.ErrCaseNotClosed,
			expectStatus: 400,
		},
		{
			name:         "caller forbidden maps to 403",
			continueErr:  casehandler.ErrCaseContinueForbidden,
			expectStatus: 403,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockCase := newTestListenHandlerWithCase(mc)
			ctx := context.Background()

			customerID := uuid.FromStringOrNil("aaaaaaaa-0005-0005-0005-000000000004")
			caseID := uuid.FromStringOrNil("aaaaaaaa-0005-0005-0005-000000000005")
			callerID := uuid.FromStringOrNil("aaaaaaaa-0005-0005-0005-000000000006")

			body, _ := json.Marshal(map[string]any{
				"customer_id":     customerID.String(),
				"caller_type":     "agent",
				"caller_id":       callerID.String(),
				"caller_is_admin": false,
			})
			req := &sock.Request{URI: "/v1/cases/" + caseID.String() + "/continue", Method: sock.RequestMethodPost, Data: body}

			mockCase.EXPECT().Continue(ctx, customerID, caseID, commonidentity.OwnerTypeAgent, callerID, false).
				Return(nil, tt.continueErr)

			res, err := h.processV1CasesIDContinuePost(ctx, req)
			if err != nil {
				t.Fatalf("processV1CasesIDContinuePost() error = %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("StatusCode = %v, want %v", res.StatusCode, tt.expectStatus)
			}
		})
	}
}

// Test_ProcessV1CasesIDNotesGetPostDelete covers the full notes
// sub-resource CRUD surface.
func Test_ProcessV1CasesIDNotesGetPostDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockCase := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-0006-0006-0006-000000000001")
	caseID := uuid.FromStringOrNil("aaaaaaaa-0006-0006-0006-000000000002")
	noteID := uuid.FromStringOrNil("aaaaaaaa-0006-0006-0006-000000000003")
	authorID := uuid.FromStringOrNil("aaaaaaaa-0006-0006-0006-000000000004")

	// GET
	getBody, _ := json.Marshal(map[string]any{"customer_id": customerID.String()})
	mockCase.EXPECT().CaseNoteListByCase(ctx, customerID, caseID).Return([]*casenote.CaseNote{}, nil)
	res, err := h.processV1CasesIDNotesGet(ctx, &sock.Request{
		URI: "/v1/cases/" + caseID.String() + "/notes", Method: sock.RequestMethodGet, Data: getBody,
	})
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("GET notes: res=%v err=%v", res, err)
	}

	// POST
	postBody, _ := json.Marshal(map[string]any{
		"customer_id": customerID.String(),
		"author_type": "agent",
		"author_id":   authorID.String(),
		"text":        "hello",
	})
	mockCase.EXPECT().CaseNoteCreate(ctx, customerID, caseID, "agent", gomock.Any(), "hello").
		Return(&casenote.CaseNote{ID: noteID}, nil)
	res, err = h.processV1CasesIDNotesPost(ctx, &sock.Request{
		URI: "/v1/cases/" + caseID.String() + "/notes", Method: sock.RequestMethodPost, Data: postBody,
	})
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("POST notes: res=%v err=%v", res, err)
	}

	// DELETE
	delBody, _ := json.Marshal(map[string]any{"customer_id": customerID.String()})
	mockCase.EXPECT().CaseNoteDelete(ctx, customerID, caseID, noteID).Return(nil)
	res, err = h.processV1CasesIDNotesIDDelete(ctx, &sock.Request{
		URI: "/v1/cases/" + caseID.String() + "/notes/" + noteID.String(), Method: sock.RequestMethodDelete, Data: delBody,
	})
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("DELETE note: res=%v err=%v", res, err)
	}
}

// Test_ProcessV1CasesIDTagsGetPostDelete covers the full tags
// sub-resource CRUD surface.
func Test_ProcessV1CasesIDTagsGetPostDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockCase := newTestListenHandlerWithCase(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("aaaaaaaa-0007-0007-0007-000000000001")
	caseID := uuid.FromStringOrNil("aaaaaaaa-0007-0007-0007-000000000002")
	tagID := uuid.FromStringOrNil("aaaaaaaa-0007-0007-0007-000000000003")

	// GET
	getBody, _ := json.Marshal(map[string]any{"customer_id": customerID.String()})
	mockCase.EXPECT().CaseTagList(ctx, customerID, caseID).Return([]uuid.UUID{tagID}, nil)
	res, err := h.processV1CasesIDTagsGet(ctx, &sock.Request{
		URI: "/v1/cases/" + caseID.String() + "/tags", Method: sock.RequestMethodGet, Data: getBody,
	})
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("GET tags: res=%v err=%v", res, err)
	}

	// POST
	postBody, _ := json.Marshal(map[string]any{"customer_id": customerID.String(), "tag_id": tagID.String()})
	mockCase.EXPECT().CaseTagAdd(ctx, customerID, caseID, tagID).Return(nil)
	res, err = h.processV1CasesIDTagsPost(ctx, &sock.Request{
		URI: "/v1/cases/" + caseID.String() + "/tags", Method: sock.RequestMethodPost, Data: postBody,
	})
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("POST tags: res=%v err=%v", res, err)
	}

	// DELETE
	delBody, _ := json.Marshal(map[string]any{"customer_id": customerID.String()})
	mockCase.EXPECT().CaseTagRemove(ctx, customerID, caseID, tagID).Return(nil)
	res, err = h.processV1CasesIDTagsIDDelete(ctx, &sock.Request{
		URI: "/v1/cases/" + caseID.String() + "/tags/" + tagID.String(), Method: sock.RequestMethodDelete, Data: delBody,
	})
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("DELETE tag: res=%v err=%v", res, err)
	}
}

// Test_ProcessRequest_RoutesCasesURIs verifies the top-level dispatcher
// correctly routes /v1/cases/* URIs to the new handlers (regression
// guard for the regex ordering, particularly /unresolved before /{id}).
func Test_ProcessRequest_RoutesCasesURIs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockCase := newTestListenHandlerWithCase(mc)

	customerID := uuid.FromStringOrNil("aaaaaaaa-0008-0008-0008-000000000001")
	caseID := uuid.FromStringOrNil("aaaaaaaa-0008-0008-0008-000000000002")
	body, _ := json.Marshal(map[string]any{"customer_id": customerID.String()})

	mockCase.EXPECT().CaseListUnresolved(gomock.Any(), customerID).Return([]*kase.Case{}, nil)
	res, err := h.processRequest(&sock.Request{URI: "/v1/cases/unresolved", Method: sock.RequestMethodGet, Data: body})
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("routing /v1/cases/unresolved: res=%v err=%v", res, err)
	}

	mockCase.EXPECT().CaseGet(gomock.Any(), customerID, caseID).Return(&kase.Case{ID: caseID}, nil)
	res, err = h.processRequest(&sock.Request{URI: "/v1/cases/" + caseID.String(), Method: sock.RequestMethodGet, Data: body})
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("routing /v1/cases/{id}: res=%v err=%v", res, err)
	}
}
