package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

// Identity values of the default authenticated agent and shared IDs used by s1 tests.
const (
	s1CustomerID = "9b7a30c4-ab4e-11ef-9068-1b1141edabd3"
	s1AgentID    = "2a2ec0ba-8004-11ec-aea5-439829c92a7c"
)

// s1Case describes one request against a handler and its expected result.
// When wantStatus is empty the request must succeed with wantCode (default 200)
// and a body exactly equal to wantBody. Otherwise the response must be the error
// envelope with the given status and reason.
type s1Case struct {
	name       string
	method     string
	path       string
	body       string
	noAuth     bool
	setup      func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity)
	wantStatus cerrors.Status
	wantReason string
	wantCode   int
	wantBody   string
}

// s1Expect registers the mock expectation returning err.
type s1Expect func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error)

func s1Identity() *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil(s1AgentID),
			CustomerID: uuid.FromStringOrNil(s1CustomerID),
		},
	})
}

// s1Run executes every case through the generated router with the mocked servicehandler.
func s1Run(t *testing.T, tests []s1Case) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}
			a := s1Identity()
			if tt.setup != nil {
				tt.setup(mockSvc, a)
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(middleware.RequestID())
			if !tt.noAuth {
				r.Use(func(c *gin.Context) {
					c.Set("auth_identity", a)
				})
			}
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if tt.wantStatus != "" {
				assertErrorResponse(t, w, tt.wantStatus, tt.wantReason)
				return
			}

			wantCode := tt.wantCode
			if wantCode == 0 {
				wantCode = http.StatusOK
			}
			if w.Code != wantCode {
				t.Errorf("status code = %d want %d; body=%s", w.Code, wantCode, w.Body.String())
			}
			if got := w.Body.String(); got != tt.wantBody {
				t.Errorf("body = %q want %q", got, tt.wantBody)
			}
		})
	}
}

func s1ID(s string) uuid.UUID {
	return uuid.FromStringOrNil(s)
}

// s1Time parses an RFC3339 timestamp, failing loudly on a bad literal.
func s1Time(s string) *time.Time {
	v, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		panic(fmt.Sprintf("s1Time: bad literal %q: %v", s, err))
	}
	return &v
}

// s1JSON marshals v to the JSON the API is expected to return.
func s1JSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

// s1Vars returns n JSON object members ("k0":"v",...) for a variables map.
func s1Vars(n int) string {
	parts := make([]string, 0, n)
	for i := 0; i < n; i++ {
		parts = append(parts, fmt.Sprintf(`"k%d":"v"`, i))
	}
	return strings.Join(parts, ",")
}

func s1PermissionError() error {
	return cerrors.PermissionDenied(commonoutline.ServiceNameCustomerManager, "S1_DENIED", "denied by test")
}

func s1NotFoundError() error {
	return cerrors.NotFound(commonoutline.ServiceNameCallManager, "S1_NOT_FOUND", "not found by test")
}

func s1ServiceUnavailable() error {
	return fmt.Errorf("%w: boom", serviceerrors.ErrServiceUnavailable)
}

// s1Unauth is a request without an auth identity; the service must not be called.
func s1Unauth(method, path, body string) s1Case {
	return s1Case{
		name:       "unauthenticated " + method + " " + path,
		method:     method,
		path:       path,
		body:       body,
		noAuth:     true,
		wantStatus: cerrors.StatusUnauthenticated,
		wantReason: "AUTHENTICATION_REQUIRED",
	}
}

// s1BadID is a request with a malformed UUID; the service must not be called.
func s1BadID(method, path, body string) s1Case {
	return s1Case{
		name:       "invalid id " + method + " " + path,
		method:     method,
		path:       path,
		body:       body,
		wantStatus: cerrors.StatusInvalidArgument,
		wantReason: "INVALID_ID",
	}
}

// s1BadJSON is a request with a malformed JSON body; the service must not be called.
func s1BadJSON(method, path string) s1Case {
	return s1Case{
		name:       "malformed json " + method + " " + path,
		method:     method,
		path:       path,
		body:       "{not json",
		wantStatus: cerrors.StatusInvalidArgument,
		wantReason: "INVALID_JSON_BODY",
	}
}

// s1PermDenied is a request whose service call fails with a permission error.
func s1PermDenied(method, path, body string, expect s1Expect) s1Case {
	return s1Case{
		name:   "permission denied " + method + " " + path,
		method: method,
		path:   path,
		body:   body,
		setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			expect(m, a, s1PermissionError())
		},
		wantStatus: cerrors.StatusPermissionDenied,
		wantReason: "S1_DENIED",
	}
}

// s1NotFound is a request whose service call fails with a not-found error.
func s1NotFound(method, path, body string, expect s1Expect) s1Case {
	return s1Case{
		name:   "not found " + method + " " + path,
		method: method,
		path:   path,
		body:   body,
		setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			expect(m, a, s1NotFoundError())
		},
		wantStatus: cerrors.StatusNotFound,
		wantReason: "S1_NOT_FOUND",
	}
}

// s1ListCases returns the common error cases of a paginated list endpoint:
// unauthenticated, an explicit page size/token forwarded to the service and
// the page size clamped to 100.
func s1ListCases(
	path string,
	expect func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string, err error),
) []s1Case {
	return []s1Case{
		s1Unauth("GET", path, ""),
		{
			name:   "explicit page size and token are forwarded to the service",
			method: "GET",
			path:   path + "?page_size=20&page_token=tok",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				expect(m, a, 20, "tok", s1PermissionError())
			},
			wantStatus: cerrors.StatusPermissionDenied,
			wantReason: "S1_DENIED",
		},
		{
			name:   "page size above max is clamped to 100",
			method: "GET",
			path:   path + "?page_size=500",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				expect(m, a, 100, "", s1ServiceUnavailable())
			},
			wantStatus: cerrors.StatusUnavailable,
			wantReason: "SERVICE_UNAVAILABLE",
		},
	}
}
