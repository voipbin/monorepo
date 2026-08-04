package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonrequesthandler "monorepo/bin-common-handler/pkg/requesthandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)

// Test_abortWithMappedStatus tests the sentinel-to-HTTP-status mapping in
// isolation, independent of whether every branch is reachable through the
// current real implementations of AuthLogin/CustomerSelfFreeze/
// CustomerSelfFreezeAndDelete/CustomerSelfRecover. As of this writing, only
// the ErrPermissionDenied/ErrDirectAccessNotSupported and ErrBadRequest
// branches are reachable through those four call sites (see the comment on
// abortWithMappedStatus) -- the rest are exercised here to guard against a
// regression in the switch itself and to be correct once the upstream
// bare-400 collapse in bin-agent-manager/bin-customer-manager is fixed.
func Test_abortWithMappedStatus(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectStatus int
	}{
		{
			name:         "serviceerrors.ErrAuthenticationRequired maps to 401",
			err:          fmt.Errorf("%w: token expired", serviceerrors.ErrAuthenticationRequired),
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "commonrequesthandler.ErrUnauthorized maps to 401",
			err:          fmt.Errorf("%w: invalid credentials", commonrequesthandler.ErrUnauthorized),
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "serviceerrors.ErrPermissionDenied maps to 403",
			err:          fmt.Errorf("%w: not customer admin", serviceerrors.ErrPermissionDenied),
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "serviceerrors.ErrDirectAccessNotSupported maps to 403",
			err:          serviceerrors.ErrDirectAccessNotSupported,
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "commonrequesthandler.ErrForbidden maps to 403",
			err:          commonrequesthandler.ErrForbidden,
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "serviceerrors.ErrNotFound maps to 404",
			err:          fmt.Errorf("%w: customer not found", serviceerrors.ErrNotFound),
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "commonrequesthandler.ErrNotFound maps to 404",
			err:          commonrequesthandler.ErrNotFound,
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "serviceerrors.ErrInvalidArgument maps to 422",
			err:          fmt.Errorf("%w: bad field", serviceerrors.ErrInvalidArgument),
			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name:         "commonrequesthandler.ErrBadRequest maps to 400 (current real behavior of the RPC-backed calls)",
			err:          commonrequesthandler.ErrBadRequest,
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "unrecognized error maps to 500",
			err:          errors.New("unexpected internal error"),
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("POST", "/", nil)

			abortWithMappedStatus(c, tt.err)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, w.Code)
			}
		})
	}
}

func setupUnregisterServer(app *gin.Engine) {
	app.POST("/auth/unregister", PostAuthUnregister)
	app.DELETE("/auth/unregister", DeleteAuthUnregister)
}

func Test_PostAuthUnregister(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validIdentity := &auth.AuthIdentity{
		Type: auth.TypeAgent,
	}

	tests := []struct {
		name         string
		body         interface{}
		setIdentity  bool
		mockSetup    func(*servicehandler.MockServiceHandler)
		expectStatus int
	}{
		{
			name:         "missing auth_identity returns 401",
			body:         map[string]string{"password": "test-password"},
			setIdentity:  false,
			mockSetup:    func(m *servicehandler.MockServiceHandler) {},
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "bad JSON body returns 400",
			body:         "not json at all",
			setIdentity:  true,
			mockSetup:    func(m *servicehandler.MockServiceHandler) {},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "neither password nor confirmation_phrase returns 400",
			body:         map[string]string{},
			setIdentity:  true,
			mockSetup:    func(m *servicehandler.MockServiceHandler) {},
			expectStatus: http.StatusBadRequest,
		},
		{
			name: "both password and confirmation_phrase returns 400",
			body: map[string]string{
				"password":            "test-password",
				"confirmation_phrase": "DELETE",
			},
			setIdentity:  true,
			mockSetup:    func(m *servicehandler.MockServiceHandler) {},
			expectStatus: http.StatusBadRequest,
		},
		{
			// AuthLogin's underlying RPC (bin-agent-manager's processV1Login)
			// currently collapses every failure -- including a wrong password
			// -- to a bare HTTP 400, which AuthLogin surfaces here as
			// commonrequesthandler.ErrBadRequest. See abortWithMappedStatus's
			// doc comment.
			name:        "wrong password returns 400 (bare RPC failure)",
			body:        map[string]string{"password": "wrong-password"},
			setIdentity: true,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().AuthLogin(gomock.Any(), gomock.Any(), "wrong-password").
					Return("", fmt.Errorf("%w: could not login the agent", commonrequesthandler.ErrBadRequest))
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "wrong confirmation phrase returns 400",
			body:         map[string]string{"confirmation_phrase": "WRONG"},
			setIdentity:  true,
			mockSetup:    func(m *servicehandler.MockServiceHandler) {},
			expectStatus: http.StatusBadRequest,
		},
		{
			// CustomerSelfFreeze returns serviceerrors.ErrPermissionDenied
			// directly from a local permission check, before any RPC call --
			// this is the one downstream-error path this fix actually changes
			// from 400 to a distinguishable status today.
			name:        "permission denied on freeze returns 403, not 400",
			body:        map[string]string{"confirmation_phrase": "DELETE"},
			setIdentity: true,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().CustomerSelfFreeze(gomock.Any(), validIdentity).
					Return(nil, fmt.Errorf("%w: not customer admin", serviceerrors.ErrPermissionDenied))
			},
			expectStatus: http.StatusForbidden,
		},
		{
			// CustomerSelfFreezeAndDelete's underlying RPC currently collapses
			// every failure (including not-found) to a bare HTTP 400, same as
			// AuthLogin. See abortWithMappedStatus's doc comment.
			name: "customer-not-found on freeze-and-delete returns 400 (bare RPC failure)",
			body: map[string]any{
				"confirmation_phrase": "DELETE",
				"immediate":           true,
			},
			setIdentity: true,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().CustomerSelfFreezeAndDelete(gomock.Any(), validIdentity).
					Return(nil, fmt.Errorf("%w: could not freeze and delete the customer", commonrequesthandler.ErrBadRequest))
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:        "internal error on freeze returns 500, not 400",
			body:        map[string]string{"confirmation_phrase": "DELETE"},
			setIdentity: true,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().CustomerSelfFreeze(gomock.Any(), validIdentity).
					Return(nil, errors.New("unexpected internal error"))
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:        "success with confirmation phrase returns 200",
			body:        map[string]string{"confirmation_phrase": "DELETE"},
			setIdentity: true,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().CustomerSelfFreeze(gomock.Any(), validIdentity).
					Return(&cscustomer.WebhookMessage{}, nil)
			},
			expectStatus: http.StatusOK,
		},
		{
			name:        "success with password returns 200",
			body:        map[string]string{"password": "correct-password"},
			setIdentity: true,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().AuthLogin(gomock.Any(), gomock.Any(), "correct-password").
					Return("jwt-token", nil)
				m.EXPECT().CustomerSelfFreeze(gomock.Any(), validIdentity).
					Return(&cscustomer.WebhookMessage{}, nil)
			},
			expectStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			tt.mockSetup(mockSvc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				if tt.setIdentity {
					c.Set("auth_identity", validIdentity)
				}
			})
			setupUnregisterServer(r)

			var bodyBytes []byte
			if s, ok := tt.body.(string); ok {
				bodyBytes = []byte(s)
			} else {
				var err error
				bodyBytes, err = json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("Failed to marshal body: %v", err)
				}
			}

			req, _ := http.NewRequest("POST", "/auth/unregister", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d (body: %s)", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}

func Test_DeleteAuthUnregister(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validIdentity := &auth.AuthIdentity{
		Type: auth.TypeAgent,
	}

	tests := []struct {
		name         string
		setIdentity  bool
		mockSetup    func(*servicehandler.MockServiceHandler)
		expectStatus int
	}{
		{
			name:         "missing auth_identity returns 401",
			setIdentity:  false,
			mockSetup:    func(m *servicehandler.MockServiceHandler) {},
			expectStatus: http.StatusUnauthorized,
		},
		{
			// CustomerSelfRecover returns serviceerrors.ErrPermissionDenied
			// directly from a local permission check, before any RPC call.
			name:        "permission denied returns 403, not 400",
			setIdentity: true,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().CustomerSelfRecover(gomock.Any(), validIdentity).
					Return(nil, fmt.Errorf("%w: not customer admin", serviceerrors.ErrPermissionDenied))
			},
			expectStatus: http.StatusForbidden,
		},
		{
			// CustomerSelfRecover's underlying RPC currently collapses every
			// failure (including not-found) to a bare HTTP 400. See
			// abortWithMappedStatus's doc comment.
			name:        "customer-not-found returns 400 (bare RPC failure)",
			setIdentity: true,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().CustomerSelfRecover(gomock.Any(), validIdentity).
					Return(nil, fmt.Errorf("%w: could not recover the customer", commonrequesthandler.ErrBadRequest))
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:        "internal error returns 500, not 400",
			setIdentity: true,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().CustomerSelfRecover(gomock.Any(), validIdentity).
					Return(nil, errors.New("unexpected internal error"))
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:        "success returns 200",
			setIdentity: true,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().CustomerSelfRecover(gomock.Any(), validIdentity).
					Return(&cscustomer.WebhookMessage{}, nil)
			},
			expectStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			tt.mockSetup(mockSvc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				if tt.setIdentity {
					c.Set("auth_identity", validIdentity)
				}
			})
			setupUnregisterServer(r)

			req, _ := http.NewRequest("DELETE", "/auth/unregister", nil)

			r.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d (body: %s)", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}
