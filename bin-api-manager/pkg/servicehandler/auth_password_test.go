package servicehandler

import (
	"context"
	"fmt"
	"testing"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_AuthPasswordForgot(t *testing.T) {

	tests := []struct {
		name string

		username string

		responseForgotErr error
	}{
		{
			name: "normal",

			username: "test@voipbin.net",

			responseForgotErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockReq.EXPECT().AgentV1PasswordForgot(ctx, 30000, tt.username).Return(tt.responseForgotErr)

			err := h.AuthPasswordForgot(ctx, tt.username)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_AuthPasswordForgot_AgentNotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler:  mockReq,
		dbHandler:   mockDB,
		utilHandler: mockUtil,
	}
	ctx := context.Background()

	mockReq.EXPECT().AgentV1PasswordForgot(ctx, 30000, "unknown@voipbin.net").Return(fmt.Errorf("not found"))

	err := h.AuthPasswordForgot(ctx, "unknown@voipbin.net")
	if err != nil {
		t.Errorf("Wrong match. expect: nil (always returns nil), got: %v", err)
	}
}

func Test_AuthPasswordReset(t *testing.T) {

	tests := []struct {
		name string

		token    string
		password string
	}{
		{
			name: "normal",

			token:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			password: "newpassword123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockReq.EXPECT().AgentV1PasswordReset(ctx, 30000, tt.token, tt.password).Return(nil)

			err := h.AuthPasswordReset(ctx, tt.token, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_AuthPasswordReset_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler:  mockReq,
		dbHandler:   mockDB,
		utilHandler: mockUtil,
	}
	ctx := context.Background()

	mockReq.EXPECT().AgentV1PasswordReset(ctx, 30000, "invalidtoken", "newpassword123").Return(fmt.Errorf("invalid token"))

	err := h.AuthPasswordReset(ctx, "invalidtoken", "newpassword123")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}
