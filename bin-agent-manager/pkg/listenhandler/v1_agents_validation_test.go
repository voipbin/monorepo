package listenhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/agenthandler"
	"monorepo/bin-agent-manager/pkg/dbhandler"
)

const (
	validationTestAgentID     = "f1ba04b0-951e-11ee-a0a2-7b8600a1ee45"
	validationTestReferenceID = "0b6f6a4a-951f-11ee-9d6c-8f2b1d6a3c11"
)

// validationHandler is a method expression of one processV1AgentsXxx function, called directly (the router never reaches the length checks).
type validationHandler func(h *listenHandler, ctx context.Context, m *sock.Request) (*sock.Response, error)

func Test_processV1AgentsGet_validation(t *testing.T) {
	notFound := cerrors.NotFound(commonoutline.ServiceNameAgentManager, "AGENT_NOT_FOUND", "The agent was not found.")

	tests := []struct {
		name string

		uri  string
		data []byte

		listErr    error
		expectList bool

		expectStatus int
		expectErr    bool
		expectTyped  *cerrors.VoipbinError
	}{
		{
			name:      "uri that cannot be parsed returns an error",
			uri:       "/v1/agents%zz",
			expectErr: true,
		},
		{
			name:         "filters that are not json are rejected with 400",
			uri:          "/v1/agents?page_size=10",
			data:         []byte(`{bad`),
			expectStatus: 400,
		},
		{
			name:         "filter value that cannot be converted is rejected with 400",
			uri:          "/v1/agents?page_size=10",
			data:         []byte(`{"customer_id":"zzz"}`),
			expectStatus: 400,
		},
		{
			name:         "typed error from the handler is passed through",
			uri:          "/v1/agents?page_size=10&page_token=2021-11-23T17:55:39.712000Z",
			data:         []byte(`{"deleted":false}`),
			listErr:      notFound,
			expectList:   true,
			expectStatus: 404,
			expectTyped:  notFound,
		},
		{
			name:         "unclassified error from the handler is 500",
			uri:          "/v1/agents?page_size=10&page_token=2021-11-23T17:55:39.712000Z",
			data:         []byte(`{"deleted":false}`),
			listErr:      fmt.Errorf("db connection lost"),
			expectList:   true,
			expectStatus: 500,
		},
		{
			name:         "legacy not found error from the handler is 404",
			uri:          "/v1/agents?page_size=10&page_token=2021-11-23T17:55:39.712000Z",
			data:         []byte(`{"deleted":false}`),
			listErr:      fmt.Errorf("wrapped: %w", dbhandler.ErrNotFound),
			expectList:   true,
			expectStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAgent := agenthandler.NewMockAgentHandler(mc)
			h := &listenHandler{agentHandler: mockAgent}

			if tt.expectList {
				mockAgent.EXPECT().List(gomock.Any(), uint64(10), "2021-11-23T17:55:39.712000Z", map[agent.Field]any{agent.FieldDeleted: false}).Return(nil, tt.listErr)
			}

			res, err := h.processV1AgentsGet(context.Background(), &sock.Request{URI: tt.uri, Method: sock.RequestMethodGet, Data: tt.data})
			assertValidationResult(t, res, err, tt.expectErr, tt.expectStatus, tt.expectTyped)
		})
	}
}

func Test_processV1AgentsID_invalid_id_validation(t *testing.T) {
	tests := []struct {
		name    string
		handler validationHandler
		method  sock.RequestMethod
	}{
		{
			name:    "get with a non uuid id",
			handler: (*listenHandler).processV1AgentsIDGet,
			method:  sock.RequestMethodGet,
		},
		{
			name:    "delete with a non uuid id",
			handler: (*listenHandler).processV1AgentsIDDelete,
			method:  sock.RequestMethodDelete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no expectations are registered, so any agentHandler call fails the test.
			h := &listenHandler{agentHandler: agenthandler.NewMockAgentHandler(mc)}

			res, err := tt.handler(h, context.Background(), &sock.Request{URI: "/v1/agents/not-a-uuid", Method: tt.method})
			assertValidationResult(t, res, err, false, 400, nil)
		})
	}
}

func Test_processV1Agents_short_uri_validation(t *testing.T) {
	short4 := "/v1/agents"
	short5 := "/v1/agents/" + validationTestAgentID

	tests := []struct {
		name    string
		handler validationHandler
		uri     string
	}{
		{"get by id", (*listenHandler).processV1AgentsIDGet, short4},
		{"delete", (*listenHandler).processV1AgentsIDDelete, short4},
		{"login", (*listenHandler).processV1AgentsUsernameLogin, short5},
		{"addresses put", (*listenHandler).processV1AgentsIDAddressesPut, short5},
		{"info put", (*listenHandler).processV1AgentsIDPut, short4},
		{"status put", (*listenHandler).processV1AgentsIDStatusPut, short5},
		{"password put", (*listenHandler).processV1AgentsIDPasswordPut, short5},
		{"tag ids put", (*listenHandler).processV1AgentsIDTagIDsPut, short5},
		{"permission put", (*listenHandler).processV1AgentsIDPermissionPut, short5},
		{"reserve post", (*listenHandler).processV1AgentsIDReservePost, short5},
		{"reserve release post", (*listenHandler).processV1AgentsIDReserveReleasePost, short5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no expectations are registered, so any agentHandler call fails the test.
			h := &listenHandler{agentHandler: agenthandler.NewMockAgentHandler(mc)}

			// the body is valid for every handler, so only the uri length can cause the 400.
			res, err := tt.handler(h, context.Background(), &sock.Request{URI: tt.uri, Data: []byte(`{}`)})
			assertValidationResult(t, res, err, false, 400, nil)
		})
	}
}

func Test_processV1Agents_malformed_body_validation(t *testing.T) {
	idURI := "/v1/agents/" + validationTestAgentID
	tests := []struct {
		name    string
		handler validationHandler
		uri     string
	}{
		{"get by customer id and address", (*listenHandler).processV1AgentsGetByCustomerIDAddressPost, "/v1/agents/get_by_customer_id_address"},
		{"login", (*listenHandler).processV1AgentsUsernameLogin, "/v1/agents/test-user/login"},
		{"addresses put", (*listenHandler).processV1AgentsIDAddressesPut, idURI + "/addresses"},
		{"info put", (*listenHandler).processV1AgentsIDPut, idURI},
		{"status put", (*listenHandler).processV1AgentsIDStatusPut, idURI + "/status"},
		{"password put", (*listenHandler).processV1AgentsIDPasswordPut, idURI + "/password"},
		{"tag ids put", (*listenHandler).processV1AgentsIDTagIDsPut, idURI + "/tag_ids"},
		{"permission put", (*listenHandler).processV1AgentsIDPermissionPut, idURI + "/permission"},
		{"reserve post", (*listenHandler).processV1AgentsIDReservePost, idURI + "/reserve"},
		{"reserve release post", (*listenHandler).processV1AgentsIDReserveReleasePost, idURI + "/reserve_release"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no expectations are registered, so any agentHandler call fails the test.
			h := &listenHandler{agentHandler: agenthandler.NewMockAgentHandler(mc)}

			res, err := tt.handler(h, context.Background(), &sock.Request{URI: tt.uri, Data: []byte(`{bad`)})
			assertValidationResult(t, res, err, false, 400, nil)
		})
	}
}

func Test_processV1Agents_handler_error_validation(t *testing.T) {
	id := uuid.FromStringOrNil(validationTestAgentID)
	refID := uuid.FromStringOrNil(validationTestReferenceID)
	idURI := "/v1/agents/" + validationTestAgentID

	notFound := cerrors.NotFound(commonoutline.ServiceNameAgentManager, "AGENT_NOT_FOUND", "The agent was not found.")

	tests := []struct {
		name    string
		handler validationHandler
		uri     string
		data    []byte

		// expect registers the failing agentHandler call.
		expect func(m *agenthandler.MockAgentHandler, err error)

		expectStatus int
		// expectTyped is set when the typed error must be passed through to the response.
		expectTyped *cerrors.VoipbinError
	}{
		{
			name:    "get by id maps the handler error",
			handler: (*listenHandler).processV1AgentsIDGet,
			uri:     idURI,
			expect: func(m *agenthandler.MockAgentHandler, err error) {
				m.EXPECT().Get(gomock.Any(), id).Return(nil, err)
			},
			expectStatus: 404,
			expectTyped:  notFound,
		},
		{
			name:    "get by customer id and address maps the handler error",
			handler: (*listenHandler).processV1AgentsGetByCustomerIDAddressPost,
			uri:     "/v1/agents/get_by_customer_id_address",
			data:    []byte(`{"customer_id":"5fd7f9b8-cb37-11ee-bd29-f30560a6ac86","address":{"type":"extension","target":"test-ext"}}`),
			expect: func(m *agenthandler.MockAgentHandler, err error) {
				m.EXPECT().GetByCustomerIDAndAddress(gomock.Any(), uuid.FromStringOrNil("5fd7f9b8-cb37-11ee-bd29-f30560a6ac86"), &commonaddress.Address{Type: commonaddress.TypeExtension, Target: "test-ext"}).Return(nil, err)
			},
			expectStatus: 404,
			expectTyped:  notFound,
		},
		{
			name:    "login handler error returns 400",
			handler: (*listenHandler).processV1AgentsUsernameLogin,
			uri:     "/v1/agents/test-user/login",
			data:    []byte(`{"password":"test-password"}`),
			expect: func(m *agenthandler.MockAgentHandler, err error) {
				m.EXPECT().Login(gomock.Any(), "test-user", "test-password").Return(nil, err)
			},
			expectStatus: 400,
		},
		{
			name:    "addresses put handler error returns 400",
			handler: (*listenHandler).processV1AgentsIDAddressesPut,
			uri:     idURI + "/addresses",
			data:    []byte(`{"addresses":[{"type":"extension","target":"test-ext"}]}`),
			expect: func(m *agenthandler.MockAgentHandler, err error) {
				m.EXPECT().UpdateAddresses(gomock.Any(), id, []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: "test-ext"}}).Return(nil, err)
			},
			expectStatus: 400,
		},
		{
			name:    "info put handler error returns 400",
			handler: (*listenHandler).processV1AgentsIDPut,
			uri:     idURI,
			data:    []byte(`{"name":"test name"}`),
			expect: func(m *agenthandler.MockAgentHandler, err error) {
				name := "test name"
				m.EXPECT().UpdateBasicInfo(gomock.Any(), id, &name, (*string)(nil), (*agent.RingMethod)(nil)).Return(nil, err)
			},
			expectStatus: 400,
		},
		{
			name:    "status put handler error returns 400",
			handler: (*listenHandler).processV1AgentsIDStatusPut,
			uri:     idURI + "/status",
			data:    []byte(`{"status":"available"}`),
			expect: func(m *agenthandler.MockAgentHandler, err error) {
				m.EXPECT().UpdateStatus(gomock.Any(), id, agent.StatusAvailable).Return(nil, err)
			},
			expectStatus: 400,
		},
		{
			name:    "password put handler error returns 400",
			handler: (*listenHandler).processV1AgentsIDPasswordPut,
			uri:     idURI + "/password",
			data:    []byte(`{"password":"test-password"}`),
			expect: func(m *agenthandler.MockAgentHandler, err error) {
				m.EXPECT().UpdatePassword(gomock.Any(), id, "test-password").Return(nil, err)
			},
			expectStatus: 400,
		},
		{
			name:    "tag ids put handler error returns 400",
			handler: (*listenHandler).processV1AgentsIDTagIDsPut,
			uri:     idURI + "/tag_ids",
			data:    []byte(`{"tag_ids":["f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a"]}`),
			expect: func(m *agenthandler.MockAgentHandler, err error) {
				m.EXPECT().UpdateTagIDs(gomock.Any(), id, []uuid.UUID{uuid.FromStringOrNil("f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a")}).Return(nil, err)
			},
			expectStatus: 400,
		},
		{
			name:    "reserve handler error returns 500",
			handler: (*listenHandler).processV1AgentsIDReservePost,
			uri:     idURI + "/reserve",
			data:    []byte(`{"reference_type":"call","reference_id":"` + validationTestReferenceID + `"}`),
			expect: func(m *agenthandler.MockAgentHandler, err error) {
				m.EXPECT().Reserve(gomock.Any(), id, "call", refID).Return(false, err)
			},
			expectStatus: 500,
		},
		{
			name:    "reserve release handler error returns 500",
			handler: (*listenHandler).processV1AgentsIDReserveReleasePost,
			uri:     idURI + "/reserve_release",
			data:    []byte(`{"reference_id":"` + validationTestReferenceID + `"}`),
			expect: func(m *agenthandler.MockAgentHandler, err error) {
				m.EXPECT().ReserveRelease(gomock.Any(), id, refID).Return(err)
			},
			expectStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAgent := agenthandler.NewMockAgentHandler(mc)
			h := &listenHandler{agentHandler: mockAgent}

			tt.expect(mockAgent, notFound)

			res, err := tt.handler(h, context.Background(), &sock.Request{URI: tt.uri, Data: tt.data})
			assertValidationResult(t, res, err, false, tt.expectStatus, tt.expectTyped)

			// a typed error must reach the response body only for the handlers that use errorResponse.
			if tt.expectTyped == nil && res != nil && res.DataType == cerrors.DataTypeVoipbinError {
				t.Errorf("Wrong data type. the typed error must not be passed through, got: %s", res.DataType)
			}
			if tt.expectTyped == nil && res != nil && len(res.Data) != 0 {
				t.Errorf("Wrong body. expect: empty, got: %s", res.Data)
			}
		})
	}
}

func assertValidationResult(t *testing.T, res *sock.Response, err error, expectErr bool, expectStatus int, expectTyped *cerrors.VoipbinError) {
	t.Helper()

	if expectErr {
		if err == nil {
			t.Fatalf("Wrong match. expect: error, got: ok")
		}
		return
	}
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res == nil {
		t.Fatalf("Wrong match. expect: a response, got: nil")
	}
	if res.StatusCode != expectStatus {
		t.Errorf("Wrong status. expect: %d, got: %d", expectStatus, res.StatusCode)
	}
	if expectTyped != nil {
		expect, errExpect := cerrors.ToResponse(expectTyped)
		if errExpect != nil {
			t.Fatalf("Could not build the expected response. err: %v", errExpect)
		}
		if !reflect.DeepEqual(res, expect) {
			t.Errorf("Wrong response.\nexpect: %s %s\ngot: %s %s", expect.DataType, expect.Data, res.DataType, res.Data)
		}
	}
}
