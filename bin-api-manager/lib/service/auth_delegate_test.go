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

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func setupDelegateServer(app *gin.Engine) {
	authGroup := app.Group("/auth")
	authGroup.POST("/delegate", PostDelegate)
}

func Test_PostDelegate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validCustomerID := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001")
	validReason := "investigating dropped call for customer"

	validIdentity := &auth.AuthIdentity{
		Type: auth.TypeAgent,
	}

	tests := []struct {
		name         string
		body         interface{}
		setIdentity  bool
		identity     *auth.AuthIdentity
		mockSetup    func(*servicehandler.MockServiceHandler)
		expectStatus int
	}{
		{
			name:         "bad JSON body returns 400",
			body:         "not json at all",
			setIdentity:  false,
			mockSetup:    func(m *servicehandler.MockServiceHandler) {},
			expectStatus: http.StatusBadRequest,
		},
		{
			name: "invalid customer_id UUID returns 422",
			body: map[string]string{
				"customer_id": "not-a-valid-uuid",
				"reason":      validReason,
			},
			setIdentity:  false,
			mockSetup:    func(m *servicehandler.MockServiceHandler) {},
			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "missing auth_identity returns 401",
			body: map[string]string{
				"customer_id": validCustomerID.String(),
				"reason":      validReason,
			},
			setIdentity:  false,
			mockSetup:    func(m *servicehandler.MockServiceHandler) {},
			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "permission denied error returns 403",
			body: map[string]string{
				"customer_id": validCustomerID.String(),
				"reason":      validReason,
			},
			setIdentity: true,
			identity:    validIdentity,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().AuthDelegate(gomock.Any(), validIdentity, validCustomerID, validReason).
					Return(nil, fmt.Errorf("%w: not superadmin", serviceerrors.ErrPermissionDenied))
			},
			expectStatus: http.StatusForbidden,
		},
		{
			name: "not found error returns 404",
			body: map[string]string{
				"customer_id": validCustomerID.String(),
				"reason":      validReason,
			},
			setIdentity: true,
			identity:    validIdentity,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().AuthDelegate(gomock.Any(), validIdentity, validCustomerID, validReason).
					Return(nil, fmt.Errorf("%w: customer not found", serviceerrors.ErrNotFound))
			},
			expectStatus: http.StatusNotFound,
		},
		{
			name: "invalid argument error returns 422",
			body: map[string]string{
				"customer_id": validCustomerID.String(),
				"reason":      validReason,
			},
			setIdentity: true,
			identity:    validIdentity,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().AuthDelegate(gomock.Any(), validIdentity, validCustomerID, validReason).
					Return(nil, fmt.Errorf("%w: reason too short", serviceerrors.ErrInvalidArgument))
			},
			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "internal error returns 500",
			body: map[string]string{
				"customer_id": validCustomerID.String(),
				"reason":      validReason,
			},
			setIdentity: true,
			identity:    validIdentity,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().AuthDelegate(gomock.Any(), validIdentity, validCustomerID, validReason).
					Return(nil, errors.New("unexpected internal error"))
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "success returns 200 with response body",
			body: map[string]string{
				"customer_id": validCustomerID.String(),
				"reason":      validReason,
			},
			setIdentity: true,
			identity:    validIdentity,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().AuthDelegate(gomock.Any(), validIdentity, validCustomerID, validReason).
					Return(&servicehandler.DelegateResponse{
						Token:      "test.jwt.token",
						CustomerID: validCustomerID,
						Expire:     "2026-05-18T13:00:00Z",
					}, nil)
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
					c.Set("auth_identity", tt.identity)
				}
			})
			setupDelegateServer(r)

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

			req, _ := http.NewRequest("POST", "/auth/delegate", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d (body: %s)", tt.expectStatus, w.Code, w.Body.String())
			}

			// For success case, verify response body has expected fields
			if tt.expectStatus == http.StatusOK {
				var res servicehandler.DelegateResponse
				if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				}
				if res.Token == "" {
					t.Errorf("Expected non-empty token in response")
				}
				if res.CustomerID != validCustomerID {
					t.Errorf("Expected customer_id %s, got %s", validCustomerID, res.CustomerID)
				}
			}
		})
	}
}
