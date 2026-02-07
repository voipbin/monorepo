package agenthandler

import (
	"context"
	"fmt"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/cachehandler"
	"monorepo/bin-agent-manager/pkg/dbhandler"
)

func Test_PasswordForgot(t *testing.T) {

	tests := []struct {
		name string

		username  string
		emailType PasswordResetEmailType

		responseAgent *agent.Agent
		responseErr   error
	}{
		{
			name: "normal - forgot password",

			username:  "test@voipbin.net",
			emailType: PasswordResetEmailTypeForgot,

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				},
				Username: "test@voipbin.net",
			},
			responseErr: nil,
		},
		{
			name: "normal - welcome email",

			username:  "new@voipbin.net",
			emailType: PasswordResetEmailTypeWelcome,

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bc810dc4-298c-11ee-984c-ebb7811c4114"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				},
				Username: "new@voipbin.net",
			},
			responseErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &agentHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				cache:         mockCache,
			}
			ctx := context.Background()

			mockDB.EXPECT().AgentGetByUsername(ctx, tt.username).Return(tt.responseAgent, tt.responseErr)
			mockCache.EXPECT().PasswordResetTokenSet(ctx, gomock.Any(), tt.responseAgent.ID, passwordResetTokenTTL).Return(nil)
			mockReq.EXPECT().EmailV1EmailSend(ctx, uuid.Nil, uuid.Nil, []commonaddress.Address{
				{Type: commonaddress.TypeEmail, Target: tt.username},
			}, gomock.Any(), gomock.Any(), gomock.Nil()).Return(nil, nil)

			err := h.PasswordForgot(ctx, tt.username, tt.emailType)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_PasswordForgot_AgentNotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &agentHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
		cache:         mockCache,
	}
	ctx := context.Background()

	mockDB.EXPECT().AgentGetByUsername(ctx, "unknown@voipbin.net").Return(nil, fmt.Errorf("not found"))

	err := h.PasswordForgot(ctx, "unknown@voipbin.net", PasswordResetEmailTypeForgot)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_PasswordForgot_CacheSetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &agentHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
		cache:         mockCache,
	}
	ctx := context.Background()

	responseAgent := &agent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
		},
		Username: "test@voipbin.net",
	}

	mockDB.EXPECT().AgentGetByUsername(ctx, "test@voipbin.net").Return(responseAgent, nil)
	mockCache.EXPECT().PasswordResetTokenSet(ctx, gomock.Any(), responseAgent.ID, passwordResetTokenTTL).Return(fmt.Errorf("cache error"))

	err := h.PasswordForgot(ctx, "test@voipbin.net", PasswordResetEmailTypeForgot)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_PasswordReset(t *testing.T) {

	tests := []struct {
		name string

		token    string
		password string

		responseAgentID uuid.UUID
		responseAgent   *agent.Agent
	}{
		{
			name: "normal",

			token:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			password: "newpassword123",

			responseAgentID: uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				},
				Username: "test@voipbin.net",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &agentHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				cache:         mockCache,
			}
			ctx := context.Background()

			mockCache.EXPECT().PasswordResetTokenGet(ctx, tt.token).Return(tt.responseAgentID, nil)
			mockUtil.EXPECT().HashGenerate(tt.password, defaultPasswordHashCost).Return("hashed_password", nil)
			mockDB.EXPECT().AgentSetPasswordHash(ctx, tt.responseAgentID, "hashed_password").Return(nil)
			mockCache.EXPECT().PasswordResetTokenDelete(ctx, tt.token).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.responseAgentID).Return(tt.responseAgent, nil)
			mockNotify.EXPECT().PublishEvent(ctx, agent.EventTypeAgentUpdated, tt.responseAgent)

			err := h.PasswordReset(ctx, tt.token, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_PasswordReset_PasswordTooShort(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &agentHandler{}

	err := h.PasswordReset(context.Background(), "sometoken", "short")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_PasswordReset_InvalidToken(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &agentHandler{
		cache: mockCache,
	}
	ctx := context.Background()

	mockCache.EXPECT().PasswordResetTokenGet(ctx, "invalidtoken").Return(uuid.Nil, fmt.Errorf("not found"))

	err := h.PasswordReset(ctx, "invalidtoken", "newpassword123")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_PasswordReset_GuestAgent(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &agentHandler{
		cache: mockCache,
	}
	ctx := context.Background()

	mockCache.EXPECT().PasswordResetTokenGet(ctx, "sometoken64charslong1234567890abcdef1234567890abcdef12345678").Return(agent.GuestAgentID, nil)

	err := h.PasswordReset(ctx, "sometoken64charslong1234567890abcdef1234567890abcdef12345678", "newpassword123")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_PasswordReset_HashError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &agentHandler{
		utilHandler: mockUtil,
		cache:       mockCache,
	}
	ctx := context.Background()

	agentID := uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114")
	mockCache.EXPECT().PasswordResetTokenGet(ctx, "validtoken").Return(agentID, nil)
	mockUtil.EXPECT().HashGenerate("newpassword123", defaultPasswordHashCost).Return("", fmt.Errorf("hash error"))

	err := h.PasswordReset(ctx, "validtoken", "newpassword123")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_PasswordReset_DBError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &agentHandler{
		utilHandler: mockUtil,
		db:          mockDB,
		cache:       mockCache,
	}
	ctx := context.Background()

	agentID := uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114")
	mockCache.EXPECT().PasswordResetTokenGet(ctx, "validtoken").Return(agentID, nil)
	mockUtil.EXPECT().HashGenerate("newpassword123", defaultPasswordHashCost).Return("hashed_password", nil)
	mockDB.EXPECT().AgentSetPasswordHash(ctx, agentID, "hashed_password").Return(fmt.Errorf("db error"))

	err := h.PasswordReset(ctx, "validtoken", "newpassword123")
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}
