package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

// s2Err is the typed service error returned by mocked service handlers in s2 tests.
var s2Err = cerrors.PermissionDenied(commonoutline.ServiceNameCustomerManager, "S2_DENIED", "denied by test")

// s2Case describes one request against a handler and the expected error envelope.
type s2Case struct {
	name       string
	method     string
	path       string
	body       string
	noAuth     bool
	identity   *auth.AuthIdentity // overrides the default agent identity when set
	wantStatus cerrors.Status
	wantReason string
	setup      func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity)
}

func s2Identity() *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			CustomerID: uuid.FromStringOrNil("9b7a30c4-ab4e-11ef-9068-1b1141edabd3"),
		},
	})
}

func s2RunCases(t *testing.T, tests []s2Case) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}
			a := s2Identity()
			if tt.identity != nil {
				a = tt.identity
			}
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

			assertErrorResponse(t, w, tt.wantStatus, tt.wantReason)
		})
	}
}
