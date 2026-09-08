package service

import (
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

// Test_PostBootRefresh_statusMapping pins the status codes the widget branches
// on. PostBoot collapses everything to 400; this handler must not, because the
// client needs to tell "boot again" (401) from "stop, wrong identity" (403)
// from "our fault" (500). Getting this wrong is invisible server-side and
// shows up as a widget that either loops or gives up.
func Test_PostBootRefresh_statusMapping(t *testing.T) {
	directIdentity := auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:        uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		AllowedResourceID: uuid.FromStringOrNil("beefbeef-0000-0000-0000-000000000001"),
	})

	tests := []struct {
		name         string
		setIdentity  bool
		identity     any
		mockSetup    func(*servicehandler.MockServiceHandler)
		expectStatus int
	}{
		{
			name:         "missing auth_identity returns 401",
			setIdentity:  false,
			mockSetup:    func(*servicehandler.MockServiceHandler) {},
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "wrong type in the identity slot returns 401",
			setIdentity:  true,
			identity:     "not an identity",
			mockSetup:    func(*servicehandler.MockServiceHandler) {},
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:        "permission denied returns 403",
			setIdentity: true,
			identity:    directIdentity,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().AuthBootRefresh(gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("%w: not a direct token", serviceerrors.ErrPermissionDenied))
			},
			expectStatus: http.StatusForbidden,
		},
		{
			name:        "authentication required returns 401",
			setIdentity: true,
			identity:    directIdentity,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().AuthBootRefresh(gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("%w: direct hash rotated", serviceerrors.ErrAuthenticationRequired))
			},
			expectStatus: http.StatusUnauthorized,
		},
		{
			// A signing fault is ours. Telling the widget to boot again cannot
			// help, so it must not look like an expired token.
			name:        "internal error returns 500",
			setIdentity: true,
			identity:    directIdentity,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().AuthBootRefresh(gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("%w: token generation failed", serviceerrors.ErrInternal))
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:        "success returns 200",
			setIdentity: true,
			identity:    directIdentity,
			mockSetup: func(m *servicehandler.MockServiceHandler) {
				m.EXPECT().AuthBootRefresh(gomock.Any(), gomock.Any()).
					Return(&servicehandler.BootResponse{Token: "tok", Type: "direct"}, nil)
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

			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				if tt.setIdentity {
					c.Set("auth_identity", tt.identity)
				}
				c.Next()
			})
			r.POST("/auth/boot/refresh", PostBootRefresh)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/auth/boot/refresh", nil))

			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d", tt.expectStatus, w.Code)
			}
		})
	}
}
