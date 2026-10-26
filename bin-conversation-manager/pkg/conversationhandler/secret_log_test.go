package conversationhandler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/internal/logtest"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
)

func Test_Hook_HookVerify_logsAndErrorsHaveNoSecret(t *testing.T) {
	const sentinel = "SENTINELsecret123"

	accountID := "e8f5795a-e6eb-11ec-bb81-c3cec34bd99c"

	tests := []struct {
		name string
		call func(h *conversationHandler) error
		// setup prepares the account handler mock
		setup func(m *accounthandler.MockAccountHandler)
	}{
		{
			name: "hook with an unparsable uri",
			call: func(h *conversationHandler) error {
				return h.Hook(context.Background(), "http://h/%zz?hub.verify_token="+sentinel, "POST", "", []byte(`{}`))
			},
			setup: func(m *accounthandler.MockAccountHandler) {},
		},
		{
			name: "hook with a verify token in the uri",
			call: func(h *conversationHandler) error {
				return h.Hook(context.Background(), "https://hook.example.com/v1.0/conversation/accounts/"+accountID+"?hub.verify_token="+sentinel, "POST", "", []byte(`{}`))
			},
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("not found"))
			},
		},
		{
			name: "hook verify with an unparsable uri",
			call: func(h *conversationHandler) error {
				_, err := h.HookVerify(context.Background(), "http://h/%zz?hub.verify_token="+sentinel, "subscribe", sentinel, "c")
				return err
			},
			setup: func(m *accounthandler.MockAccountHandler) {},
		},
		{
			name: "hook verify with a short path and a verify token",
			call: func(h *conversationHandler) error {
				_, err := h.HookVerify(context.Background(), "https://hook.example.com/short?hub.verify_token="+sentinel, "subscribe", sentinel, "c")
				return err
			},
			setup: func(m *accounthandler.MockAccountHandler) {},
		},
		{
			name: "hook verify with a verify token in the uri",
			call: func(h *conversationHandler) error {
				_, err := h.HookVerify(context.Background(), "https://hook.example.com/v1.0/conversation/accounts/"+accountID+"?hub.verify_token="+sentinel, "subscribe", sentinel, "c")
				return err
			},
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("not found"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			hook := logtest.Capture(t)

			mockAccount := accounthandler.NewMockAccountHandler(mc)
			tt.setup(mockAccount)

			h := &conversationHandler{accountHandler: mockAccount}

			err := tt.call(h)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if strings.Contains(err.Error(), sentinel) {
				t.Errorf("The returned error carries the verify token: %v", err)
			}

			logtest.AssertNoLeak(t, hook, sentinel)
		})
	}
}
