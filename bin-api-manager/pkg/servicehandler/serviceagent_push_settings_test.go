package servicehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	pmnotification "monorepo/bin-push-manager/models/notification"
	pmsettings "monorepo/bin-push-manager/models/settings"
	tkchat "monorepo/bin-talk-manager/models/chat"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

var (
	psTestCustomerID      = uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	psTestOtherCustomerID = uuid.FromStringOrNil("6a1b3c4d-8e5f-11ee-97b2-cfe7337b701c")
	psTestAgentID         = uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")
	psTestChatID          = uuid.FromStringOrNil("5d7f9b1c-3e4a-4c68-8a0e-2c4e6a8c0e11")
)

func psTestAgent(perm amagent.Permission) *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity:   commonidentity.Identity{ID: psTestAgentID, CustomerID: psTestCustomerID},
		Permission: perm,
	})
}

func psTestSettings() *pmsettings.Settings {
	return pmsettings.New([]pmnotification.Kind{pmnotification.KindCaseNote}, []uuid.UUID{psTestChatID})
}

// the caller that is not an agent, an agent without an id and an agent with a permission failure stay out
func psDeniedCases() []struct {
	name   string
	agent  *auth.AuthIdentity
	expect error
} {
	return []struct {
		name   string
		agent  *auth.AuthIdentity
		expect error
	}{
		{"access key identity is not an agent", auth.NewDirectIdentity(&auth.DirectScope{CustomerID: psTestCustomerID}), serviceerrors.ErrAuthenticationRequired},
		{"agent with a nil id", auth.NewAgentIdentity(&amagent.Agent{Identity: commonidentity.Identity{CustomerID: psTestCustomerID}, Permission: amagent.PermissionCustomerAgent}), serviceerrors.ErrAuthenticationRequired},
	}
}

func Test_ServiceAgentPushSettingsGet(t *testing.T) {
	rpcErr := errors.New("rpc failed")

	t.Run("the owner comes from the identity only", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &serviceHandler{reqHandler: mockReq}
		ctx := context.Background()

		// an agent without permission bits is still an agent
		mockReq.EXPECT().PushV1SettingsGet(ctx, psTestCustomerID, psTestAgentID).Return(psTestSettings(), nil)

		res, err := h.ServiceAgentPushSettingsGet(ctx, psTestAgent(amagent.PermissionNone))
		if err != nil || !reflect.DeepEqual(res, psTestSettings()) {
			t.Errorf("Wrong match. got: %v, %v", res, err)
		}
	})

	for _, tt := range psDeniedCases() {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h := &serviceHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

			res, err := h.ServiceAgentPushSettingsGet(context.Background(), tt.agent)
			if !errors.Is(err, tt.expect) || res != nil {
				t.Errorf("Wrong match. expect: %v, got: %v, %v", tt.expect, res, err)
			}
		})
	}

	t.Run("rpc error is propagated", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &serviceHandler{reqHandler: mockReq}

		mockReq.EXPECT().PushV1SettingsGet(gomock.Any(), psTestCustomerID, psTestAgentID).Return(nil, rpcErr)
		if _, err := h.ServiceAgentPushSettingsGet(context.Background(), psTestAgent(amagent.PermissionCustomerAgent)); !errors.Is(err, rpcErr) {
			t.Errorf("Wrong match. expect: %v, got: %v", rpcErr, err)
		}
	})
}

func Test_ServiceAgentPushSettingsUpdate(t *testing.T) {
	kinds := map[pmnotification.Kind]bool{pmnotification.KindCaseNote: false}

	t.Run("ok", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &serviceHandler{reqHandler: mockReq}
		ctx := context.Background()

		mockReq.EXPECT().PushV1SettingsUpdate(ctx, psTestCustomerID, psTestAgentID, kinds).Return(psTestSettings(), nil)

		res, err := h.ServiceAgentPushSettingsUpdate(ctx, psTestAgent(amagent.PermissionCustomerAgent), kinds)
		if err != nil || !reflect.DeepEqual(res, psTestSettings()) {
			t.Errorf("Wrong match. got: %v, %v", res, err)
		}
	})

	for _, tt := range psDeniedCases() {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h := &serviceHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

			res, err := h.ServiceAgentPushSettingsUpdate(context.Background(), tt.agent, kinds)
			if !errors.Is(err, tt.expect) || res != nil {
				t.Errorf("Wrong match. expect: %v, got: %v, %v", tt.expect, res, err)
			}
		})
	}

	// the typed 400 of push-manager (a kind that can not be switched) reaches the caller unchanged, so the
	// client gets a 400 with the reason and not a 500
	t.Run("a typed error of push-manager is returned as it is", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &serviceHandler{reqHandler: mockReq}

		badKind := cerrors.InvalidArgument(commonoutline.ServiceNamePushManager, "INVALID_KIND", "x")
		mockReq.EXPECT().PushV1SettingsUpdate(gomock.Any(), psTestCustomerID, psTestAgentID, gomock.Any()).Return(nil, badKind)

		_, err := h.ServiceAgentPushSettingsUpdate(context.Background(), psTestAgent(amagent.PermissionCustomerAgent), map[pmnotification.Kind]bool{"call_incoming": false})
		var ve *cerrors.VoipbinError
		if !errors.As(err, &ve) || ve.Status != cerrors.StatusInvalidArgument || ve.Reason != "INVALID_KIND" {
			t.Errorf("Wrong match. expect: the typed 400, got: %v", err)
		}
	})
}

func Test_ServiceAgentPushMuteAdd(t *testing.T) {
	rpcErr := errors.New("rpc failed")
	chat := func(customerID uuid.UUID) *tkchat.Chat {
		return &tkchat.Chat{Identity: commonidentity.Identity{ID: psTestChatID, CustomerID: customerID}}
	}

	tests := []struct {
		name           string
		chat           *tkchat.Chat
		chatErr        error
		expectMute     bool
		muteErr        error
		expectNotFound bool
		expectErr      error
	}{
		{name: "a chat of the customer", chat: chat(psTestCustomerID), expectMute: true},
		{name: "a chat of another customer is not found", chat: chat(psTestOtherCustomerID), expectNotFound: true},
		{name: "a chat that can not be read is not found", chatErr: rpcErr, expectNotFound: true},
		{name: "a missing chat is not found", chat: nil, expectNotFound: true},
		{name: "the mute fails", chat: chat(psTestCustomerID), expectMute: true, muteErr: rpcErr, expectErr: rpcErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			mockReq.EXPECT().TalkV1ChatGet(ctx, psTestChatID).Return(tt.chat, tt.chatErr)
			if tt.expectMute {
				var res *pmsettings.Settings
				if tt.muteErr == nil {
					res = psTestSettings()
				}
				mockReq.EXPECT().PushV1MuteAdd(ctx, psTestCustomerID, psTestAgentID, psTestChatID).Return(res, tt.muteErr)
			}

			res, err := h.ServiceAgentPushMuteAdd(ctx, psTestAgent(amagent.PermissionCustomerAgent), psTestChatID)
			switch {
			case tt.expectNotFound:
				var ve *cerrors.VoipbinError
				if !errors.As(err, &ve) || ve.Status != cerrors.StatusNotFound || ve.Reason != "CHAT_NOT_FOUND" || res != nil {
					t.Errorf("Wrong match. expect: CHAT_NOT_FOUND, got: %v, %v", res, err)
				}
			case tt.expectErr != nil:
				if !errors.Is(err, tt.expectErr) || res != nil {
					t.Errorf("Wrong match. expect: %v, got: %v, %v", tt.expectErr, res, err)
				}
			default:
				if err != nil || !reflect.DeepEqual(res, psTestSettings()) {
					t.Errorf("Wrong match. got: %v, %v", res, err)
				}
			}
		})
	}

	for _, tt := range psDeniedCases() {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h := &serviceHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

			res, err := h.ServiceAgentPushMuteAdd(context.Background(), tt.agent, psTestChatID)
			if !errors.Is(err, tt.expect) || res != nil {
				t.Errorf("Wrong match. expect: %v, got: %v, %v", tt.expect, res, err)
			}
		})
	}
}

func Test_ServiceAgentPushMuteRemove(t *testing.T) {
	rpcErr := errors.New("rpc failed")

	t.Run("no chat lookup, the call is idempotent", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc) // TalkV1ChatGet has no expectation
		h := &serviceHandler{reqHandler: mockReq}
		ctx := context.Background()

		mockReq.EXPECT().PushV1MuteRemove(ctx, psTestCustomerID, psTestAgentID, psTestChatID).Return(psTestSettings(), nil)

		res, err := h.ServiceAgentPushMuteRemove(ctx, psTestAgent(amagent.PermissionCustomerAgent), psTestChatID)
		if err != nil || !reflect.DeepEqual(res, psTestSettings()) {
			t.Errorf("Wrong match. got: %v, %v", res, err)
		}
	})

	t.Run("rpc error is propagated", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &serviceHandler{reqHandler: mockReq}

		mockReq.EXPECT().PushV1MuteRemove(gomock.Any(), psTestCustomerID, psTestAgentID, psTestChatID).Return(nil, rpcErr)
		if _, err := h.ServiceAgentPushMuteRemove(context.Background(), psTestAgent(amagent.PermissionCustomerAgent), psTestChatID); !errors.Is(err, rpcErr) {
			t.Errorf("Wrong match. expect: %v, got: %v", rpcErr, err)
		}
	})

	for _, tt := range psDeniedCases() {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h := &serviceHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

			res, err := h.ServiceAgentPushMuteRemove(context.Background(), tt.agent, psTestChatID)
			if !errors.Is(err, tt.expect) || res != nil {
				t.Errorf("Wrong match. expect: %v, got: %v, %v", tt.expect, res, err)
			}
		})
	}
}
