package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	amai "monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetAiModels(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		responseModels []*amai.ModelInfo

		expectedCode int
		expectedRes  string
	}{
		{
			name: "2 items, own_or_default and custom",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			responseModels: []*amai.ModelInfo{
				{
					ID:              "gemini.gemini-2.5-flash",
					Label:           "Gemini 2.5 Flash",
					Vendor:          "Google",
					Recommended:     true,
					Tags:            []string{"low-cost"},
					Description:     "fast",
					PlatformManaged: false,
					KeyMode:         "own_or_default",
				},
				{
					ID:            "custom.openrouter",
					Label:         "OpenRouter model (your OpenRouter key)",
					Vendor:        "OpenRouter",
					Tags:          []string{},
					Description:   "Enter any model supported by OpenRouter.",
					KeyMode:       "own_required",
					ModelIDPrefix: "openrouter.",
				},
			},

			expectedCode: http.StatusOK,
			expectedRes:  `{"result":[{"id":"gemini.gemini-2.5-flash","label":"Gemini 2.5 Flash","vendor":"Google","recommended":true,"tags":["low-cost"],"description":"fast","platform_managed":false,"key_mode":"own_or_default"},{"id":"custom.openrouter","label":"OpenRouter model (your OpenRouter key)","vendor":"OpenRouter","recommended":false,"tags":[],"description":"Enter any model supported by OpenRouter.","platform_managed":false,"key_mode":"own_required","model_id_prefix":"openrouter."}],"next_page_token":""}`,
		},
		{
			name: "empty list serializes as an empty array",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			responseModels: nil,

			expectedCode: http.StatusOK,
			expectedRes:  `{"result":[],"next_page_token":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", "/ai_models", nil)
			mockSvc.EXPECT().AIModelList(req.Context(), tt.agent).Return(tt.responseModels, nil)

			r.ServeHTTP(w, req)
			if w.Code != tt.expectedCode {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectedCode, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_GetAiModels_noAuth(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest("GET", "/ai_models", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusUnauthorized, w.Code)
	}
}

func Test_GetAiModels_serviceError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")},
	})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest("GET", "/ai_models", nil)
	mockSvc.EXPECT().AIModelList(req.Context(), agent).Return(nil, errors.New("boom"))

	r.ServeHTTP(w, req)
	if w.Code == http.StatusOK {
		t.Errorf("Wrong match. expect an error status, got: %d", w.Code)
	}
}
