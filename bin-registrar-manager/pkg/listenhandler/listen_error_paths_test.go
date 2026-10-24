package listenhandler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/pkg/contacthandler"
	"monorepo/bin-registrar-manager/pkg/extensionhandler"
	"monorepo/bin-registrar-manager/pkg/trunkhandler"
)

var errInjectedErrCase = errors.New("injected failure")

const (
	errCaseID    = "e0000000-0000-4000-8000-000000000001"
	errCaseNilID = "00000000-0000-0000-0000-000000000000"
)

type listenErrFixture struct {
	h *listenHandler

	extension *extensionhandler.MockExtensionHandler
	trunk     *trunkhandler.MockTrunkHandler
	contact   *contacthandler.MockContactHandler
}

func newListenErrFixture(mc *gomock.Controller) *listenErrFixture {
	f := &listenErrFixture{
		extension: extensionhandler.NewMockExtensionHandler(mc),
		trunk:     trunkhandler.NewMockTrunkHandler(mc),
		contact:   contacthandler.NewMockContactHandler(mc),
	}
	f.h = &listenHandler{
		extensionHandler: f.extension,
		trunkHandler:     f.trunk,
		contactHandler:   f.contact,
	}

	return f
}

type listenErrCase struct {
	name string

	// call invokes the handler method under test. A mock without an expectation
	// fails the test when it is called, so each case lists exactly the calls it allows.
	call  func(h *listenHandler, ctx context.Context, req *sock.Request) (*sock.Response, error)
	req   *sock.Request
	setup func(f *listenErrFixture)

	// expectStatus is the status of the returned response. When expectErr is true the
	// handler is expected to return (nil, errInjectedErrCase) instead.
	expectStatus int
	expectErr    bool
}

func runListenErrCases(t *testing.T, tests []listenErrCase) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			f := newListenErrFixture(mc)
			if tt.setup != nil {
				tt.setup(f)
			}

			res, err := tt.call(f.h, context.Background(), tt.req)

			if tt.expectErr {
				if err == nil {
					t.Fatalf("Wrong match. expect: error, got: nil (response: %v)", res)
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil response, got: %v", res)
				}
				// a case that injects a handler failure must get that same failure back,
				// not an unrelated error.
				if tt.setup != nil && !errors.Is(err, errInjectedErrCase) {
					t.Errorf("Wrong match. expect: injected error, got: %v", err)
				}
				// a case without a handler failure sends a malformed body and must get
				// the decoding error back.
				var syntaxErr *json.SyntaxError
				if tt.setup == nil && !errors.As(err, &syntaxErr) {
					t.Errorf("Wrong match. expect: a JSON syntax error, got: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: status %d, got: %v", tt.expectStatus, res)
			}
		})
	}
}

func Test_listenhandler_extensions_failure_errors(t *testing.T) {
	notFound := cerrors.NotFound(commonoutline.ServiceNameRegistrarManager, "CASE_NOT_FOUND", "case message")

	postExt := func(h *listenHandler, ctx context.Context, r *sock.Request) (*sock.Response, error) {
		return h.processV1ExtensionsPost(ctx, r)
	}
	getExt := func(h *listenHandler, ctx context.Context, r *sock.Request) (*sock.Response, error) {
		return h.processV1ExtensionsGet(ctx, r)
	}
	putExt := func(h *listenHandler, ctx context.Context, r *sock.Request) (*sock.Response, error) {
		return h.processV1ExtensionsIDPut(ctx, r)
	}
	delExt := func(h *listenHandler, ctx context.Context, r *sock.Request) (*sock.Response, error) {
		return h.processV1ExtensionsIDDelete(ctx, r)
	}
	getByExt := func(h *listenHandler, ctx context.Context, r *sock.Request) (*sock.Response, error) {
		return h.processV1ExtensionsExtensionExtensionGet(ctx, r)
	}

	tests := []listenErrCase{
		{
			name:         "post with a malformed body is rejected before the handler is called",
			call:         postExt,
			req:          &sock.Request{URI: "/v1/extensions", Method: sock.RequestMethodPost, Data: []byte(`{"name":`)},
			expectStatus: 400,
		},
		{
			name: "post with a handler error maps to the generic failure status",
			call: postExt,
			req: &sock.Request{URI: "/v1/extensions", Method: sock.RequestMethodPost,
				Data: []byte(`{"customer_id":"` + errCaseID + `","extension":"1001","password":"case-password"}`)},
			setup: func(f *listenErrFixture) {
				f.extension.EXPECT().Create(gomock.Any(), uuid.FromStringOrNil(errCaseID), "", "", "1001", "case-password").Return(nil, errInjectedErrCase)
			},
			expectStatus: 500,
		},
		{
			name: "post with a typed handler error keeps the typed status",
			call: postExt,
			req: &sock.Request{URI: "/v1/extensions", Method: sock.RequestMethodPost,
				Data: []byte(`{"customer_id":"` + errCaseID + `","extension":"1001","password":"case-password"}`)},
			setup: func(f *listenErrFixture) {
				f.extension.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, notFound)
			},
			expectStatus: 404,
		},
		{
			name:         "list with a malformed body is rejected",
			call:         getExt,
			req:          &sock.Request{URI: "/v1/extensions?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":`)},
			expectStatus: 400,
		},
		{
			name:         "list with a filter value of the wrong type is rejected",
			call:         getExt,
			req:          &sock.Request{URI: "/v1/extensions?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":"not-a-uuid"}`)},
			expectStatus: 400,
		},
		{
			name: "list with a handler error is returned as an error",
			call: getExt,
			req:  &sock.Request{URI: "/v1/extensions?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":"` + errCaseID + `"}`)},
			setup: func(f *listenErrFixture) {
				f.extension.EXPECT().List(gomock.Any(), "", uint64(10), gomock.Any()).Return(nil, errInjectedErrCase)
			},
			expectErr: true,
		},
		{
			name:         "put with a malformed body is rejected",
			call:         putExt,
			req:          &sock.Request{URI: "/v1/extensions/" + errCaseID, Method: sock.RequestMethodPut, Data: []byte(`{"name":`)},
			expectStatus: 400,
		},
		{
			name: "put with an update error is returned as an error and the extension is not re-read",
			call: putExt,
			req:  &sock.Request{URI: "/v1/extensions/" + errCaseID, Method: sock.RequestMethodPut, Data: []byte(`{"name":"renamed"}`)},
			setup: func(f *listenErrFixture) {
				f.extension.EXPECT().Update(gomock.Any(), uuid.FromStringOrNil(errCaseID), gomock.Any()).Return(nil, errInjectedErrCase)
			},
			expectErr: true,
		},
		{
			name: "put with an error on the re-read is returned as an error",
			call: putExt,
			req:  &sock.Request{URI: "/v1/extensions/" + errCaseID, Method: sock.RequestMethodPut, Data: []byte(`{"name":"renamed"}`)},
			setup: func(f *listenErrFixture) {
				f.extension.EXPECT().Update(gomock.Any(), uuid.FromStringOrNil(errCaseID), gomock.Any()).Return(nil, nil)
				f.extension.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(errCaseID)).Return(nil, errInjectedErrCase)
			},
			expectErr: true,
		},
		{
			name: "delete with a handler error is returned as an error",
			call: delExt,
			req:  &sock.Request{URI: "/v1/extensions/" + errCaseID, Method: sock.RequestMethodDelete},
			setup: func(f *listenErrFixture) {
				f.extension.EXPECT().Delete(gomock.Any(), uuid.FromStringOrNil(errCaseID)).Return(nil, errInjectedErrCase)
			},
			expectErr: true,
		},
		{
			name:      "get by extension with a malformed body is returned as an error",
			call:      getByExt,
			req:       &sock.Request{URI: "/v1/extensions/extension/1001", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":`)},
			expectErr: true,
		},
		{
			name: "get by extension with a handler error is returned as an error",
			call: getByExt,
			req: &sock.Request{URI: "/v1/extensions/extension/1001", Method: sock.RequestMethodGet,
				Data: []byte(`{"customer_id":"` + errCaseID + `"}`)},
			setup: func(f *listenErrFixture) {
				f.extension.EXPECT().GetByExtension(gomock.Any(), uuid.FromStringOrNil(errCaseID), "1001").Return(nil, errInjectedErrCase)
			},
			expectErr: true,
		},
	}

	runListenErrCases(t, tests)
}

func Test_listenhandler_trunks_failure_errors(t *testing.T) {
	notFound := cerrors.NotFound(commonoutline.ServiceNameRegistrarManager, "CASE_NOT_FOUND", "case message")

	postTrunk := func(h *listenHandler, ctx context.Context, r *sock.Request) (*sock.Response, error) {
		return h.processV1TrunksPost(ctx, r)
	}
	getTrunks := func(h *listenHandler, ctx context.Context, r *sock.Request) (*sock.Response, error) {
		return h.processV1TrunksGet(ctx, r)
	}
	getTrunk := func(h *listenHandler, ctx context.Context, r *sock.Request) (*sock.Response, error) {
		return h.processV1TrunksIDGet(ctx, r)
	}
	putTrunk := func(h *listenHandler, ctx context.Context, r *sock.Request) (*sock.Response, error) {
		return h.processV1TrunksIDPut(ctx, r)
	}
	delTrunk := func(h *listenHandler, ctx context.Context, r *sock.Request) (*sock.Response, error) {
		return h.processV1TrunksIDDelete(ctx, r)
	}
	getByDomain := func(h *listenHandler, ctx context.Context, r *sock.Request) (*sock.Response, error) {
		return h.processV1TrunksDomainNameDomainNameGet(ctx, r)
	}

	tests := []listenErrCase{
		{
			name:         "post with a malformed body is rejected before the handler is called",
			call:         postTrunk,
			req:          &sock.Request{URI: "/v1/trunks", Method: sock.RequestMethodPost, Data: []byte(`{"name":`)},
			expectStatus: 400,
		},
		{
			name: "post with a handler error maps to the generic failure status",
			call: postTrunk,
			req:  &sock.Request{URI: "/v1/trunks", Method: sock.RequestMethodPost, Data: []byte(`{"customer_id":"` + errCaseID + `","domain_name":"case-trunk"}`)},
			setup: func(f *listenErrFixture) {
				f.trunk.EXPECT().Create(gomock.Any(), uuid.FromStringOrNil(errCaseID), "", "", "case-trunk", gomock.Any(), "", "", gomock.Any()).Return(nil, errInjectedErrCase)
			},
			expectStatus: 500,
		},
		{
			name:         "list with a malformed body is rejected",
			call:         getTrunks,
			req:          &sock.Request{URI: "/v1/trunks?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":`)},
			expectStatus: 400,
		},
		{
			name:         "list with a filter value of the wrong type is rejected",
			call:         getTrunks,
			req:          &sock.Request{URI: "/v1/trunks?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":"not-a-uuid"}`)},
			expectStatus: 400,
		},
		{
			name: "list with a handler error maps to the generic failure status",
			call: getTrunks,
			req:  &sock.Request{URI: "/v1/trunks?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":"` + errCaseID + `"}`)},
			setup: func(f *listenErrFixture) {
				f.trunk.EXPECT().List(gomock.Any(), "", uint64(10), gomock.Any()).Return(nil, errInjectedErrCase)
			},
			expectStatus: 500,
		},
		{
			name:         "get with the nil id is rejected",
			call:         getTrunk,
			req:          &sock.Request{URI: "/v1/trunks/" + errCaseNilID, Method: sock.RequestMethodGet},
			expectStatus: 400,
		},
		{
			name: "get with a typed handler error keeps the typed status",
			call: getTrunk,
			req:  &sock.Request{URI: "/v1/trunks/" + errCaseID, Method: sock.RequestMethodGet},
			setup: func(f *listenErrFixture) {
				f.trunk.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(errCaseID)).Return(nil, notFound)
			},
			expectStatus: 404,
		},
		{
			name:         "put with the nil id is rejected",
			call:         putTrunk,
			req:          &sock.Request{URI: "/v1/trunks/" + errCaseNilID, Method: sock.RequestMethodPut, Data: []byte(`{"name":"renamed"}`)},
			expectStatus: 400,
		},
		{
			name:         "put with a malformed body is rejected",
			call:         putTrunk,
			req:          &sock.Request{URI: "/v1/trunks/" + errCaseID, Method: sock.RequestMethodPut, Data: []byte(`{"name":`)},
			expectStatus: 400,
		},
		{
			name: "put with an update error maps to the generic failure status",
			call: putTrunk,
			req:  &sock.Request{URI: "/v1/trunks/" + errCaseID, Method: sock.RequestMethodPut, Data: []byte(`{"name":"renamed"}`)},
			setup: func(f *listenErrFixture) {
				f.trunk.EXPECT().Update(gomock.Any(), uuid.FromStringOrNil(errCaseID), gomock.Any()).Return(nil, errInjectedErrCase)
			},
			expectStatus: 500,
		},
		{
			name: "put without fields reads the trunk and maps a read error",
			call: putTrunk,
			req:  &sock.Request{URI: "/v1/trunks/" + errCaseID, Method: sock.RequestMethodPut, Data: []byte(`{}`)},
			setup: func(f *listenErrFixture) {
				f.trunk.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(errCaseID)).Return(nil, errInjectedErrCase)
			},
			expectStatus: 500,
		},
		{
			name:         "delete with the nil id is rejected",
			call:         delTrunk,
			req:          &sock.Request{URI: "/v1/trunks/" + errCaseNilID, Method: sock.RequestMethodDelete},
			expectStatus: 400,
		},
		{
			name: "delete with a handler error maps to the generic failure status",
			call: delTrunk,
			req:  &sock.Request{URI: "/v1/trunks/" + errCaseID, Method: sock.RequestMethodDelete},
			setup: func(f *listenErrFixture) {
				f.trunk.EXPECT().Delete(gomock.Any(), uuid.FromStringOrNil(errCaseID)).Return(nil, errInjectedErrCase)
			},
			expectStatus: 500,
		},
		{
			name: "get by domain name with a handler error is returned as an error",
			call: getByDomain,
			req:  &sock.Request{URI: "/v1/trunks/domain_name/case-trunk", Method: sock.RequestMethodGet},
			setup: func(f *listenErrFixture) {
				f.trunk.EXPECT().GetByDomainName(gomock.Any(), "case-trunk").Return(nil, errInjectedErrCase)
			},
			expectErr: true,
		},
	}

	runListenErrCases(t, tests)
}

func Test_listenhandler_count_failure_errors(t *testing.T) {
	countExt := func(h *listenHandler, ctx context.Context, r *sock.Request) (*sock.Response, error) {
		return h.processV1ExtensionsCountByCustomerGet(ctx, r)
	}
	countTrunk := func(h *listenHandler, ctx context.Context, r *sock.Request) (*sock.Response, error) {
		return h.processV1TrunksCountByCustomerGet(ctx, r)
	}

	tests := []listenErrCase{
		{
			name:         "extension count with a malformed body is rejected",
			call:         countExt,
			req:          &sock.Request{URI: "/v1/extensions/count_by_customer", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":`)},
			expectStatus: 400,
		},
		{
			name: "extension count with a handler error maps to the generic failure status",
			call: countExt,
			req:  &sock.Request{URI: "/v1/extensions/count_by_customer", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":"` + errCaseID + `"}`)},
			setup: func(f *listenErrFixture) {
				f.extension.EXPECT().CountByCustomerID(gomock.Any(), uuid.FromStringOrNil(errCaseID)).Return(0, errInjectedErrCase)
			},
			expectStatus: 500,
		},
		{
			name:         "trunk count with a malformed body is rejected",
			call:         countTrunk,
			req:          &sock.Request{URI: "/v1/trunks/count_by_customer", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":`)},
			expectStatus: 400,
		},
		{
			name: "trunk count with a handler error maps to the generic failure status",
			call: countTrunk,
			req:  &sock.Request{URI: "/v1/trunks/count_by_customer", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":"` + errCaseID + `"}`)},
			setup: func(f *listenErrFixture) {
				f.trunk.EXPECT().CountByCustomerID(gomock.Any(), uuid.FromStringOrNil(errCaseID)).Return(0, errInjectedErrCase)
			},
			expectStatus: 500,
		},
	}

	runListenErrCases(t, tests)
}

func Test_listenhandler_contacts_failure_errors(t *testing.T) {
	getContacts := func(h *listenHandler, ctx context.Context, r *sock.Request) (*sock.Response, error) {
		return h.processV1ContactsGet(ctx, r)
	}
	putContacts := func(h *listenHandler, ctx context.Context, r *sock.Request) (*sock.Response, error) {
		return h.processV1ContactsPut(ctx, r)
	}

	body := []byte(`{"customer_id":"` + errCaseID + `","extension":"1001"}`)

	tests := []listenErrCase{
		{
			name:      "get with a malformed body is returned as an error",
			call:      getContacts,
			req:       &sock.Request{URI: "/v1/contacts", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":`)},
			expectErr: true,
		},
		{
			name: "get with a handler error is returned as an error",
			call: getContacts,
			req:  &sock.Request{URI: "/v1/contacts", Method: sock.RequestMethodGet, Data: body},
			setup: func(f *listenErrFixture) {
				f.contact.EXPECT().ContactGetsByExtension(gomock.Any(), uuid.FromStringOrNil(errCaseID), "1001").Return(nil, errInjectedErrCase)
			},
			expectErr: true,
		},
		{
			name:      "put with a malformed body is returned as an error",
			call:      putContacts,
			req:       &sock.Request{URI: "/v1/contacts", Method: sock.RequestMethodPut, Data: []byte(`{"customer_id":`)},
			expectErr: true,
		},
		{
			name: "put with a handler error is returned as an error",
			call: putContacts,
			req:  &sock.Request{URI: "/v1/contacts", Method: sock.RequestMethodPut, Data: body},
			setup: func(f *listenErrFixture) {
				f.contact.EXPECT().ContactRefreshByEndpoint(gomock.Any(), uuid.FromStringOrNil(errCaseID), "1001").Return(errInjectedErrCase)
			},
			expectErr: true,
		},
	}

	runListenErrCases(t, tests)
}
