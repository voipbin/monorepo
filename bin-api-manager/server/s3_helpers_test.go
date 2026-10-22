package server

import (
	"encoding/json"
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

// s3Case is one HTTP request against the real route table with a mocked
// service handler. Either wantStatus/wantReason (error envelope) or
// wantCode/wantBody (success) is asserted.
type s3Case struct {
	name   string
	method string
	path   string
	body   string
	noAuth bool
	setup  func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity)

	wantStatus cerrors.Status
	wantReason string

	wantCode int
	wantBody string
}

func s3Identity() *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("00000000-0000-4000-8000-0000000000f1"),
			CustomerID: uuid.FromStringOrNil("00000000-0000-4000-8000-0000000000c1"),
		},
	})
}

func s3SvcErr() error {
	return cerrors.NotFound(commonoutline.ServiceNameAPIManager, "S3_SVC_NOT_FOUND", "The resource was not found.")
}

func s3JSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func s3Run(t *testing.T, cases []s3Case) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}
			a := s3Identity()

			r := gin.New()
			r.Use(middleware.RequestID())
			r.Use(func(c *gin.Context) {
				if !tt.noAuth {
					c.Set("auth_identity", a)
				}
			})
			openapi_server.RegisterHandlers(r, h)

			if tt.setup != nil {
				tt.setup(mockSvc, a)
			}

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if tt.wantStatus != "" {
				assertErrorResponse(t, w, tt.wantStatus, tt.wantReason)
				return
			}
			if w.Code != tt.wantCode {
				t.Errorf("status code = %d want %d; body=%s", w.Code, tt.wantCode, w.Body.String())
			}
			if w.Body.String() != tt.wantBody {
				t.Errorf("body mismatch.\nwant: %s\ngot:  %s", tt.wantBody, w.Body.String())
			}
		})
	}
}

var (
	s3UUID1 = uuid.FromStringOrNil("00000000-0000-4000-8000-0000000000a1")
	s3UUID2 = uuid.FromStringOrNil("00000000-0000-4000-8000-0000000000a2")
	s3UUID3 = uuid.FromStringOrNil("00000000-0000-4000-8000-0000000000a3")
)
