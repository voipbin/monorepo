package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

const (
	s4ID1 = "7c6a1f10-2b3d-4e5f-8a9b-0c1d2e3f4a51"
	s4ID2 = "7c6a1f10-2b3d-4e5f-8a9b-0c1d2e3f4a52"
	s4ID3 = "7c6a1f10-2b3d-4e5f-8a9b-0c1d2e3f4a53"
)

func s4UUID(s string) uuid.UUID {
	return uuid.FromStringOrNil(s)
}

func s4Agent() *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("c96bf1c2-a2e9-11ec-a8e3-a716ee72ed9d"),
			CustomerID: uuid.FromStringOrNil("5f7a1d3c-9e8b-4c2a-b6d4-3a1f0e9c8b77"),
		},
	})
}

// s4Case is a single request/response scenario against the generated router.
type s4Case struct {
	name   string
	method string
	path   string
	body   string
	noAuth bool

	// setup registers the exact RPC expectation. nil means the servicehandler
	// must not be called at all (any call fails the test through gomock).
	setup func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity)

	// expected success response
	wantStatus int
	wantBody   any // marshaled and compared structurally; nil skips body check
	// wantHeaders are exact response header values that must be present.
	wantHeaders map[string]string

	// expected error response (used when wantReason != "")
	wantErrStatus cerrors.Status
	wantReason    string
}

func s4Run(t *testing.T, tt s4Case) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	agent := s4Agent()
	if tt.setup != nil {
		tt.setup(mockSvc, agent)
	}

	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	if !tt.noAuth {
		r.Use(func(c *gin.Context) {
			c.Set("auth_identity", agent)
		})
	}
	openapi_server.RegisterHandlers(r, h)

	var req *http.Request
	if tt.body == "" {
		req, _ = http.NewRequest(tt.method, tt.path, nil)
	} else {
		req, _ = http.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
		req.Header.Set("Content-Type", "application/json")
	}
	r.ServeHTTP(w, req)

	if tt.wantReason != "" {
		assertErrorResponse(t, w, tt.wantErrStatus, tt.wantReason)
		return
	}

	if w.Code != tt.wantStatus {
		t.Fatalf("status = %d want %d; body=%s", w.Code, tt.wantStatus, w.Body.String())
	}
	for k, v := range tt.wantHeaders {
		if got := w.Header().Get(k); got != v {
			t.Errorf("header %s = %q want %q", k, got, v)
		}
	}
	if tt.wantBody == nil {
		return
	}
	s4CompareJSON(t, w.Body.Bytes(), tt.wantBody)
}

// s4CompareJSON compares the response body with the JSON form of want,
// structurally (key order independent).
func s4CompareJSON(t *testing.T, gotRaw []byte, want any) {
	t.Helper()
	wantRaw, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal want: %v", err)
	}
	var gotV, wantV any
	if err := json.Unmarshal(gotRaw, &gotV); err != nil {
		t.Fatalf("unmarshal got: %v; body=%s", err, string(gotRaw))
	}
	if err := json.Unmarshal(wantRaw, &wantV); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	if !reflect.DeepEqual(gotV, wantV) {
		t.Errorf("body mismatch.\ngot : %s\nwant: %s", string(gotRaw), string(wantRaw))
	}
}

// s4Obj builds a zero-valued T whose ID and TMCreate fields (when present) are
// populated, so that responses are distinguishable.
func s4Obj[T any](id string, tm *time.Time) *T {
	v := new(T)
	rv := reflect.ValueOf(v).Elem()
	if f := rv.FieldByName("ID"); f.IsValid() && f.CanSet() && f.Type() == reflect.TypeOf(uuid.Nil) {
		f.Set(reflect.ValueOf(uuid.FromStringOrNil(id)))
	}
	if f := rv.FieldByName("TMCreate"); f.IsValid() && f.CanSet() && f.Type() == reflect.TypeOf(tm) {
		f.Set(reflect.ValueOf(tm))
	}
	return v
}

func s4Time(s string) *time.Time {
	tm, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		panic(err)
	}
	return &tm
}

// s4Errs are the reusable service errors; the assertion side maps them through
// translateToVoipbinError.
var (
	s4ErrDenied = serviceerrors.ErrPermissionDenied
)

// s4ItemEP describes an authenticated endpoint operating on one resource (or on
// none) so the common 401 / invalid-id / bad-json / service-error / success
// matrix can be generated.
type s4ItemEP struct {
	name   string
	method string
	path   string // valid request path
	body   string // valid request body ("" = none)

	badIDPath   string // path with a malformed id ("" = skip)
	badIDReason string

	badJSON bool // body is JSON that must be rejected when malformed

	// call returns the expectation for the valid request.
	call func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call
	ret  any // typed object returned on success
	// zero is returned together with the error (default untyped nil; set for
	// non-nillable return types such as string).
	zero any

	okStatus int
	// wantBody overrides the expected body (defaults to ret)
	wantBody any
}

func s4ItemCases(ep s4ItemEP) []s4Case {
	var cases []s4Case

	cases = append(cases, s4Case{
		name: ep.name + "/unauthenticated", method: ep.method, path: ep.path, body: ep.body, noAuth: true,
		wantErrStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED",
	})

	if ep.badIDPath != "" {
		cases = append(cases, s4Case{
			name: ep.name + "/invalid_id", method: ep.method, path: ep.badIDPath, body: ep.body,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: ep.badIDReason,
		})
	}

	if ep.badJSON {
		cases = append(cases, s4Case{
			name: ep.name + "/bad_json", method: ep.method, path: ep.path, body: "{not json",
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY",
		})
	}

	cases = append(cases, s4Case{
		name: ep.name + "/service_error", method: ep.method, path: ep.path, body: ep.body,
		setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			ep.call(m, a).Return(ep.zero, s4ErrDenied)
		},
		wantErrStatus: cerrors.StatusPermissionDenied, wantReason: "PERMISSION_DENIED",
	})

	want := ep.wantBody
	if want == nil {
		want = ep.ret
	}
	cases = append(cases, s4Case{
		name: ep.name + "/success", method: ep.method, path: ep.path, body: ep.body,
		setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			ep.call(m, a).Return(ep.ret, nil)
		},
		wantStatus: ep.okStatus, wantBody: want,
	})

	return cases
}

// s4ListEP describes a paginated list endpoint.
type s4ListEP struct {
	name string
	path string // base path, optionally with a fixed query ("?x=y")
	// sep is "?" if path has no query yet, "&" otherwise.
	sep string

	// call returns the expectation for the given page size/token.
	call func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call
	// mk builds the typed []*T the service returns; tms are the TMCreate values.
	mk func(tms ...*time.Time) any
	// itemsJSON returns the structure expected in "result" for the given mk output
	// (the slice itself is marshaled).
}

func s4ListCases(ep s4ListEP) []s4Case {
	sep := ep.sep
	if sep == "" {
		sep = "?"
	}
	tm := s4Time("2024-03-05T10:20:30.123456Z")

	listBody := func(items any, token string) any {
		return map[string]any{"result": items, "next_page_token": token}
	}

	sizeCase := func(name, query string, wantSize uint64, wantToken string) s4Case {
		return s4Case{
			name: ep.name + "/" + name, method: http.MethodGet, path: ep.path + query,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				ep.call(m, a, wantSize, wantToken).Return(ep.mk(), nil)
			},
			wantStatus: 200, wantBody: listBody([]any{}, ""),
		}
	}

	items := ep.mk(tm)
	return []s4Case{
		{
			name: ep.name + "/unauthenticated", method: http.MethodGet, path: ep.path, noAuth: true,
			wantErrStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED",
		},
		sizeCase("default_size_100", "", 100, ""),
		sizeCase("size_zero_falls_back_to_100", sep+"page_size=0", 100, ""),
		sizeCase("size_over_limit_clamped_to_100", sep+"page_size=101", 100, ""),
		sizeCase("size_and_token_passthrough", sep+"page_size=7&page_token=tok-1", 7, "tok-1"),
		{
			name: ep.name + "/service_error", method: http.MethodGet, path: ep.path,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				ep.call(m, a, 100, "").Return(nil, s4ErrDenied)
			},
			wantErrStatus: cerrors.StatusPermissionDenied, wantReason: "PERMISSION_DENIED",
		},
		{
			name: ep.name + "/next_token_from_last_tm_create", method: http.MethodGet, path: ep.path,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				ep.call(m, a, 100, "").Return(items, nil)
			},
			wantStatus: 200, wantBody: listBody(items, "2024-03-05T10:20:30.123456Z"),
		},
		{
			name: ep.name + "/nil_tm_create_yields_empty_token", method: http.MethodGet, path: ep.path,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				ep.call(m, a, 100, "").Return(ep.mk((*time.Time)(nil)), nil)
			},
			wantStatus: 200, wantBody: listBody(ep.mk((*time.Time)(nil)), ""),
		},
	}
}

func s4RunCases(t *testing.T, cases []s4Case) {
	t.Helper()
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s4Run(t, tt)
		})
	}
}

// s4MkList builds a []*T whose TMCreate values are tms.
func s4MkList[T any](tms ...*time.Time) any {
	ids := []string{s4ID1, s4ID2, s4ID3}
	out := []*T{}
	for i, tm := range tms {
		out = append(out, s4Obj[T](ids[i%len(ids)], tm))
	}
	return out
}
