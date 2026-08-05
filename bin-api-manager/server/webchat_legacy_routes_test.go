package server

// End-to-end regression coverage for VOIP-1311: the legacy underscore-named
// webchat_* routes (/webchat_widgets, /webchat_sessions, /webchat_messages)
// registered manually in cmd/api-manager/main.go alongside the generated
// hyphenated routes (/webchat-widgets, /webchat-sessions, /webchat-messages).
//
// This file replicates main.go's actual v1 group wiring (RateLimit +
// Authenticate + CustomerRateLimit + EnforceAccountStatus +
// RegisterHandlersWithOptions + the legacy alias block) so these tests
// exercise the real middleware chain, not a stand-in. Prometheus
// counter behavior for the deprecation marker is unit-tested directly in
// lib/middleware/webchat_deprecation_test.go; this file only asserts the
// response headers it sets, as a proxy for "the marker ran". See
// docs/plans/2026-08-05-voip-1311-webchat-resource-path-rename-design.md.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	modelscommon "monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cscustomer "monorepo/bin-customer-manager/models/customer"
	wcmessage "monorepo/bin-webchat-manager/models/message"
	wcsession "monorepo/bin-webchat-manager/models/session"
	wcwidget "monorepo/bin-webchat-manager/models/widget"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

// newWebchatLegacyRoutesTestRouter builds a gin router that mirrors the
// relevant slice of cmd/api-manager/main.go's route wiring: a single v1
// group carrying RateLimit + Authenticate + CustomerRateLimit +
// EnforceAccountStatus (VOIP-1302 order), the generated hyphenated routes
// via RegisterHandlersWithOptions, and the 15 manually-registered legacy
// underscore-named webchat_* aliases via the generated
// ServerInterfaceWrapper, exactly as main.go does it.
//
// CustomerRateLimit is wired with an all-zero CustomerRateLimitConfig, which
// per its documented "set to 0 to disable" convention (see
// lib/middleware/customer_ratelimit.go) disables every tier without ever
// touching the RateLimiter -- so a nil ratelimithandler.RateLimiter is safe
// here and no Redis/mock double is needed. This keeps the test focused on
// the IP-based RateLimit tier (the only one under test in this file) while
// still exercising the real production middleware order, in particular that
// EnforceAccountStatus() -- not Authenticate() -- is what triggers the
// CustomerRawSelfGet frozen-account check mocked by mockValidAuth.
//
// rps/burst are caller-controlled so the rate-limit test can use a very
// small budget without affecting the other tests in this file.
func newWebchatLegacyRoutesTestRouter(mockSH *servicehandler.MockServiceHandler, rps float64, burst int) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(modelscommon.OBJServiceHandler, mockSH)
		c.Next()
	})

	appServer := &server{serviceHandler: mockSH}

	v1 := r.Group("v1.0")
	v1.Use(middleware.RateLimit("v1", rps, burst))
	v1.Use(middleware.Authenticate())
	v1.Use(middleware.CustomerRateLimit(nil, middleware.CustomerRateLimitConfig{}))
	v1.Use(middleware.EnforceAccountStatus())

	openapi_server.RegisterHandlersWithOptions(v1, appServer, openapi_server.GinServerOptions{
		ErrorHandler: BindingErrorHandler,
	})

	// Mirrors the "legacy webchat_* route aliases" block in main.go.
	legacyWrapper := openapi_server.ServerInterfaceWrapper{
		Handler:      appServer,
		ErrorHandler: BindingErrorHandler,
	}
	v1.GET("/webchat_widgets", middleware.WebchatDeprecationMarker(), legacyWrapper.GetWebchatWidgets)
	v1.POST("/webchat_widgets", middleware.WebchatDeprecationMarker(), legacyWrapper.PostWebchatWidgets)
	v1.GET("/webchat_widgets/:id", middleware.WebchatDeprecationMarker(), legacyWrapper.GetWebchatWidgetsId)
	v1.PUT("/webchat_widgets/:id", middleware.WebchatDeprecationMarker(), legacyWrapper.PutWebchatWidgetsId)
	v1.DELETE("/webchat_widgets/:id", middleware.WebchatDeprecationMarker(), legacyWrapper.DeleteWebchatWidgetsId)
	v1.POST("/webchat_widgets/:id/direct_hash_regenerate", middleware.WebchatDeprecationMarker(), legacyWrapper.PostWebchatWidgetsIdDirectHashRegenerate)

	v1.GET("/webchat_sessions", middleware.WebchatDeprecationMarker(), legacyWrapper.GetWebchatSessions)
	v1.POST("/webchat_sessions", middleware.WebchatDeprecationMarker(), legacyWrapper.PostWebchatSessions)
	v1.GET("/webchat_sessions/:id", middleware.WebchatDeprecationMarker(), legacyWrapper.GetWebchatSessionsId)
	v1.DELETE("/webchat_sessions/:id", middleware.WebchatDeprecationMarker(), legacyWrapper.DeleteWebchatSessionsId)
	v1.POST("/webchat_sessions/:id/end", middleware.WebchatDeprecationMarker(), legacyWrapper.PostWebchatSessionsIdEnd)

	v1.GET("/webchat_messages", middleware.WebchatDeprecationMarker(), legacyWrapper.GetWebchatMessages)
	v1.POST("/webchat_messages", middleware.WebchatDeprecationMarker(), legacyWrapper.PostWebchatMessages)
	v1.GET("/webchat_messages/:id", middleware.WebchatDeprecationMarker(), legacyWrapper.GetWebchatMessagesId)
	v1.DELETE("/webchat_messages/:id", middleware.WebchatDeprecationMarker(), legacyWrapper.DeleteWebchatMessagesId)

	return r
}

// webchatTestAgent is a stand-in authenticated CustomerAdmin agent used
// across this file's tests.
var webchatTestAgent = amagent.Agent{
	Identity: commonidentity.Identity{
		ID:         uuid.FromStringOrNil("f1a1b2c3-1111-4a4a-9a9a-000000000001"),
		CustomerID: uuid.FromStringOrNil("f1a1b2c3-2222-4a4a-9a9a-000000000002"),
	},
	Permission: amagent.PermissionCustomerAdmin,
}

// mockValidAuth wires the two ServiceHandler calls the Authenticate
// middleware makes for a valid, non-frozen agent token: AuthJWTParse to
// decode the bearer token, then CustomerRawSelfGet for the frozen-account
// check.
func mockValidAuth(mockSH *servicehandler.MockServiceHandler, token string) {
	mockSH.EXPECT().AuthJWTParse(gomock.Any(), token).Return(map[string]interface{}{
		"agent": webchatTestAgent,
	}, nil)
	mockSH.EXPECT().CustomerRawSelfGet(gomock.Any(), gomock.Any()).Return(&cscustomer.WebhookMessage{
		Status: cscustomer.StatusActive,
	}, nil)
}

// TestWebchatLegacyRoutes_WorkEndToEnd proves each of the 15 legacy
// underscore-named routes reaches the same handler as its hyphenated
// counterpart and returns 200, not just that the router matches the path.
func TestWebchatLegacyRoutes_WorkEndToEnd(t *testing.T) {
	widgetID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-000000000001")
	sessionID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-000000000002")
	messageID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-000000000003")

	tests := []struct {
		name      string
		method    string
		path      string
		body      string
		mockSetup func(mockSH *servicehandler.MockServiceHandler)
	}{
		{
			name:   "GET /webchat_widgets",
			method: http.MethodGet,
			path:   "/v1.0/webchat_widgets",
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().WebchatWidgetList(gomock.Any(), gomock.Any(), uint64(100), "").Return([]*wcwidget.WebhookMessage{}, nil)
			},
		},
		{
			name:   "POST /webchat_widgets",
			method: http.MethodPost,
			path:   "/v1.0/webchat_widgets",
			body:   `{"name":"Support","session_flow_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}`,
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().WebchatWidgetCreate(gomock.Any(), gomock.Any(), "Support", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&wcwidget.WebhookMessage{}, nil)
			},
		},
		{
			name:   "GET /webchat_widgets/{id}",
			method: http.MethodGet,
			path:   "/v1.0/webchat_widgets/" + widgetID.String(),
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().WebchatWidgetGet(gomock.Any(), gomock.Any(), widgetID).Return(&wcwidget.WebhookMessage{}, nil)
			},
		},
		{
			name:   "PUT /webchat_widgets/{id}",
			method: http.MethodPut,
			path:   "/v1.0/webchat_widgets/" + widgetID.String(),
			body:   `{"name":"Support 2","session_flow_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}`,
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().WebchatWidgetUpdate(gomock.Any(), gomock.Any(), widgetID, "Support 2", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&wcwidget.WebhookMessage{}, nil)
			},
		},
		{
			name:   "DELETE /webchat_widgets/{id}",
			method: http.MethodDelete,
			path:   "/v1.0/webchat_widgets/" + widgetID.String(),
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().WebchatWidgetDelete(gomock.Any(), gomock.Any(), widgetID).Return(&wcwidget.WebhookMessage{}, nil)
			},
		},
		{
			name:   "POST /webchat_widgets/{id}/direct_hash_regenerate",
			method: http.MethodPost,
			path:   "/v1.0/webchat_widgets/" + widgetID.String() + "/direct_hash_regenerate",
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().WebchatWidgetDirectHashRegenerate(gomock.Any(), gomock.Any(), widgetID).Return(&wcwidget.WebhookMessage{}, nil)
			},
		},
		{
			name:   "GET /webchat_sessions",
			method: http.MethodGet,
			path:   "/v1.0/webchat_sessions",
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().WebchatSessionList(gomock.Any(), gomock.Any(), uint64(100), "", gomock.Any()).Return([]*wcsession.WebhookMessage{}, nil)
			},
		},
		{
			name:   "POST /webchat_sessions",
			method: http.MethodPost,
			path:   "/v1.0/webchat_sessions",
			body:   `{"widget_id":"` + widgetID.String() + `"}`,
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().WebchatSessionCreate(gomock.Any(), gomock.Any(), widgetID, gomock.Any(), gomock.Any()).Return(&wcsession.WebhookMessage{}, nil)
			},
		},
		{
			name:   "GET /webchat_sessions/{id}",
			method: http.MethodGet,
			path:   "/v1.0/webchat_sessions/" + sessionID.String(),
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().WebchatSessionGet(gomock.Any(), gomock.Any(), sessionID).Return(&wcsession.WebhookMessage{}, nil)
			},
		},
		{
			name:   "DELETE /webchat_sessions/{id}",
			method: http.MethodDelete,
			path:   "/v1.0/webchat_sessions/" + sessionID.String(),
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().WebchatSessionDelete(gomock.Any(), gomock.Any(), sessionID).Return(&wcsession.WebhookMessage{}, nil)
			},
		},
		{
			name:   "POST /webchat_sessions/{id}/end",
			method: http.MethodPost,
			path:   "/v1.0/webchat_sessions/" + sessionID.String() + "/end",
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().WebchatSessionEnd(gomock.Any(), gomock.Any(), sessionID).Return(&wcsession.WebhookMessage{}, nil)
			},
		},
		{
			name:   "GET /webchat_messages",
			method: http.MethodGet,
			path:   "/v1.0/webchat_messages",
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().WebchatMessageList(gomock.Any(), gomock.Any(), uint64(100), "", gomock.Any()).Return([]*wcmessage.WebhookMessage{}, nil)
			},
		},
		{
			name:   "POST /webchat_messages",
			method: http.MethodPost,
			path:   "/v1.0/webchat_messages",
			body:   `{"session_id":"` + sessionID.String() + `","direction":"outbound","text":"hi"}`,
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().WebchatMessageCreate(gomock.Any(), gomock.Any(), sessionID, gomock.Any(), "hi").Return(&wcmessage.WebhookMessage{}, nil)
			},
		},
		{
			name:   "GET /webchat_messages/{id}",
			method: http.MethodGet,
			path:   "/v1.0/webchat_messages/" + messageID.String(),
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().WebchatMessageGet(gomock.Any(), gomock.Any(), messageID).Return(&wcmessage.WebhookMessage{}, nil)
			},
		},
		{
			name:   "DELETE /webchat_messages/{id}",
			method: http.MethodDelete,
			path:   "/v1.0/webchat_messages/" + messageID.String(),
			mockSetup: func(mockSH *servicehandler.MockServiceHandler) {
				mockSH.EXPECT().WebchatMessageDelete(gomock.Any(), gomock.Any(), messageID).Return(&wcmessage.WebhookMessage{}, nil)
			},
		},
	}

	if len(tests) != 15 {
		t.Fatalf("expected exactly 15 legacy webchat_* routes under test, got %d -- update this table if the legacy route set changed", len(tests))
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSH := servicehandler.NewMockServiceHandler(mc)
			mockValidAuth(mockSH, "validToken")
			tt.mockSetup(mockSH)

			r := newWebchatLegacyRoutesTestRouter(mockSH, 1000, 1000)

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			req.Header.Set("Authorization", "Bearer validToken")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d want 200; body=%s", w.Code, w.Body.String())
			}

			// Every legacy response carries the deprecation markers.
			if got := w.Header().Get("Deprecation"); got != "true" {
				t.Errorf("Deprecation header = %q want %q", got, "true")
			}
			if got := w.Header().Get("Sunset"); got == "" {
				t.Errorf("Sunset header is empty, want an RFC 8594 HTTP-date")
			}
		})
	}
}

// TestWebchatLegacyRoutes_RejectUnauthenticated proves the legacy routes
// inherit the v1 group's Authenticate middleware: an unauthenticated
// request must be rejected with 401 before the handler (and therefore the
// mock ServiceHandler) is ever reached.
func TestWebchatLegacyRoutes_RejectUnauthenticated(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSH := servicehandler.NewMockServiceHandler(mc)
	// No AuthJWTParse/CustomerRawSelfGet/Webchat* expectations configured:
	// gomock fails the test if any of them are called.

	r := newWebchatLegacyRoutesTestRouter(mockSH, 1000, 1000)

	req := httptest.NewRequest(http.MethodGet, "/v1.0/webchat_widgets", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d want %d; body=%s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}

// TestWebchatNewRoutes_RejectUnauthenticated is the hyphenated-route
// counterpart to the above, confirming both route sets enforce auth
// identically.
func TestWebchatNewRoutes_RejectUnauthenticated(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSH := servicehandler.NewMockServiceHandler(mc)

	r := newWebchatLegacyRoutesTestRouter(mockSH, 1000, 1000)

	req := httptest.NewRequest(http.MethodGet, "/v1.0/webchat-widgets", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d want %d; body=%s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}

// TestWebchatNewRoutes_WorkEndToEnd is the regression counterpart to
// TestWebchatLegacyRoutes_WorkEndToEnd: proves the new hyphenated routes
// still work correctly and do NOT carry the deprecation markers.
func TestWebchatNewRoutes_WorkEndToEnd(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSH := servicehandler.NewMockServiceHandler(mc)
	mockValidAuth(mockSH, "validToken")
	mockSH.EXPECT().WebchatWidgetList(gomock.Any(), gomock.Any(), uint64(100), "").Return([]*wcwidget.WebhookMessage{}, nil)

	r := newWebchatLegacyRoutesTestRouter(mockSH, 1000, 1000)

	req := httptest.NewRequest(http.MethodGet, "/v1.0/webchat-widgets", nil)
	req.Header.Set("Authorization", "Bearer validToken")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d want 200; body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Deprecation"); got != "" {
		t.Errorf("Deprecation header = %q, want empty on the new hyphenated route", got)
	}
	if got := w.Header().Get("Sunset"); got != "" {
		t.Errorf("Sunset header = %q, want empty on the new hyphenated route", got)
	}
}

// TestWebchatLegacyRoutes_RateLimitSharedWithNewRoutes proves the legacy
// routes are rate-limited by the SAME per-IP v1 tier bucket as the new
// routes (they share the v1 group's RateLimit middleware instance), not a
// separate, more permissive bucket.
func TestWebchatLegacyRoutes_RateLimitSharedWithNewRoutes(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSH := servicehandler.NewMockServiceHandler(mc)
	// Only ONE request is expected to reach a handler: the rest must be
	// rejected by the rate limiter before auth/handler logic runs.
	mockValidAuth(mockSH, "validToken")
	mockSH.EXPECT().WebchatWidgetList(gomock.Any(), gomock.Any(), uint64(100), "").Return([]*wcwidget.WebhookMessage{}, nil)

	// burst=1: the first request in the tier consumes the single token: any
	// further request from the same client IP is rejected until refill.
	r := newWebchatLegacyRoutesTestRouter(mockSH, 0.001, 1)

	// First request against the NEW route consumes the shared bucket's only
	// token.
	req1 := httptest.NewRequest(http.MethodGet, "/v1.0/webchat-widgets", nil)
	req1.Header.Set("Authorization", "Bearer validToken")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first (new-route) request status = %d want 200; body=%s", w1.Code, w1.Body.String())
	}

	// Second request, against the LEGACY route, from the same IP: must be
	// rate-limited (429) because it shares the same v1 tier bucket -- if the
	// legacy routes had their own bucket this would incorrectly succeed.
	req2 := httptest.NewRequest(http.MethodGet, "/v1.0/webchat_widgets", nil)
	req2.Header.Set("Authorization", "Bearer validToken")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("second (legacy-route) request status = %d want %d (rate limited); body=%s", w2.Code, http.StatusTooManyRequests, w2.Body.String())
	}
}
