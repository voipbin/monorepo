package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)

// okCase describes one request that is expected to reach the service layer and
// return a JSON body with the given status.
type okCase struct {
	name       string
	method     string
	path       string
	body       string
	wantStatus int
	wantJSON   string // expected response body, compared structurally
	// wantFields, when set, is compared against the same top-level keys of the response body.
	wantFields map[string]any
	setup      func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity)
}

func okRunCases(t *testing.T, tests []okCase) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}
			a := s2Identity()
			if tt.setup != nil {
				tt.setup(mockSvc, a)
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(middleware.RequestID())
			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", a)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d. body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantJSON == "" && tt.wantFields == nil {
				t.Fatal("okCase must set wantJSON or wantFields")
			}
			if tt.wantFields != nil {
				var got map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
					t.Fatalf("response is not a JSON object: %v. body: %s", err, w.Body.String())
				}
				for k, want := range tt.wantFields {
					if !reflect.DeepEqual(got[k], want) {
						t.Errorf("field %q = %#v, want %#v", k, got[k], want)
					}
				}
			}
			if tt.wantJSON == "" {
				return
			}

			var got, want any
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("response is not JSON: %v. body: %s", err, w.Body.String())
			}
			if err := json.Unmarshal([]byte(tt.wantJSON), &want); err != nil {
				t.Fatalf("bad wantJSON: %v", err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("body mismatch.\ngot:  %s\nwant: %s", w.Body.String(), tt.wantJSON)
			}
		})
	}
}
