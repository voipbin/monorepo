package listenhandler

import (
	"fmt"
	"testing"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/internal/logtest"
	"monorepo/bin-conversation-manager/models/account"
)

const (
	sentinelSecret = "SENTINELsecret123"
	sentinelToken  = "SENTINELtoken456"
)

func Test_logsHaveNoSecret(t *testing.T) {
	accountID := uuid.FromStringOrNil("792c0222-e4a9-11ec-af5e-679fe5991907")
	hookURI := "https://hook.example.com/v1.0/conversation/accounts/" + accountID.String() + "?hub.mode=subscribe&hub.verify_token=" + sentinelSecret + "&hub.challenge=c"

	tests := []struct {
		name string

		request *sock.Request
		setup   func(m *listenMocks)
	}{
		{
			name:    "accounts post fails in the handler",
			request: newReq("/v1/accounts", sock.RequestMethodPost, `{"customer_id":"6d2e4b63-6e8f-4f1d-8a7b-1b2c3d4e5f60","type":"line","secret":"`+sentinelSecret+`","token":"`+sentinelToken+`"}`),
			setup: func(m *listenMocks) {
				m.acc.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), sentinelSecret, sentinelToken, gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("could not setup"))
			},
		},
		{
			name:    "accounts post unmarshal failure",
			request: newReq("/v1/accounts", sock.RequestMethodPost, `{"secret":"`+sentinelSecret+`","token":"`+sentinelToken+`",`),
			setup:   func(m *listenMocks) {},
		},
		{
			name:    "accounts post marshal failure",
			request: newReq("/v1/accounts", sock.RequestMethodPost, `{"secret":"`+sentinelSecret+`"}`),
			setup: func(m *listenMocks) {
				m.acc.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&account.Account{Secret: sentinelSecret, Token: sentinelToken, ProviderData: []byte("{bad")}, nil)
			},
		},
		{
			name:    "accounts get marshal failure",
			request: newReq(testAccURI, sock.RequestMethodGet, ""),
			setup: func(m *listenMocks) {
				m.acc.EXPECT().Get(gomock.Any(), accountID).
					Return(&account.Account{Secret: sentinelSecret, Token: sentinelToken, ProviderData: []byte("{bad")}, nil)
			},
		},
		{
			name:    "accounts list marshal failure",
			request: newReq("/v1/accounts?page_size=10", sock.RequestMethodGet, `{"deleted":false}`),
			setup: func(m *listenMocks) {
				m.acc.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]*account.Account{{Secret: sentinelSecret, Token: sentinelToken, ProviderData: []byte("{bad")}}, nil)
			},
		},
		{
			name:    "accounts put carries a new secret",
			request: newReq(testAccURI, sock.RequestMethodPut, `{"secret":"`+sentinelSecret+`","token":"`+sentinelToken+`"}`),
			setup: func(m *listenMocks) {
				m.acc.EXPECT().Update(gomock.Any(), accountID, gomock.Any()).
					Return(&account.Account{Secret: sentinelSecret, Token: sentinelToken, ProviderData: []byte("{bad")}, nil)
			},
		},
		{
			name:    "accounts put fails",
			request: newReq(testAccURI, sock.RequestMethodPut, `{"secret":"`+sentinelSecret+`"}`),
			setup: func(m *listenMocks) {
				m.acc.EXPECT().Update(gomock.Any(), accountID, gomock.Any()).Return(nil, fmt.Errorf("could not update"))
			},
		},
		{
			name:    "accounts delete marshal failure",
			request: newReq(testAccURI, sock.RequestMethodDelete, ""),
			setup: func(m *listenMocks) {
				m.acc.EXPECT().Delete(gomock.Any(), accountID).
					Return(&account.Account{Secret: sentinelSecret, Token: sentinelToken, ProviderData: []byte("{bad")}, nil)
			},
		},
		{
			name:    "not found",
			request: newReq("/v1/unknown", sock.RequestMethodPost, `{"secret":"`+sentinelSecret+`"}`),
			setup:   func(m *listenMocks) {},
		},
		{
			name:    "hooks get with a verify token",
			request: newReq("/v1/hooks", sock.RequestMethodGet, `{"received_uri":"`+hookURI+`"}`),
			setup: func(m *listenMocks) {
				m.conv.EXPECT().HookVerify(gomock.Any(), hookURI, "subscribe", sentinelSecret, "c").Return("", fmt.Errorf("verify failed"))
			},
		},
		{
			name:    "hooks get with an unparsable uri",
			request: newReq("/v1/hooks", sock.RequestMethodGet, `{"received_uri":"http://h/%zz?hub.verify_token=`+sentinelSecret+`"}`),
			setup:   func(m *listenMocks) {},
		},
		{
			name:    "hooks post with a verify token in the uri",
			request: newReq("/v1/hooks", sock.RequestMethodPost, `{"received_uri":"`+hookURI+`","received_method":"POST"}`),
			setup: func(m *listenMocks) {
				m.conv.EXPECT().Hook(gomock.Any(), hookURI, gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("hook failed"))
			},
		},
		{
			name:    "hooks post unmarshal failure",
			request: newReq("/v1/hooks", sock.RequestMethodPost, `{"received_uri":"`+hookURI+`",`),
			setup:   func(m *listenMocks) {},
		},
		{
			name:    "messages get carries the sentinel in the body",
			request: newReq("/v1/messages?page_size=10", sock.RequestMethodGet, `{"secret":"`+sentinelSecret+`"}`),
			setup: func(m *listenMocks) {
				m.msg.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("list failed"))
			},
		},
		{
			name:    "messages post unmarshal failure",
			request: newReq("/v1/messages", sock.RequestMethodPost, `{"text":"`+sentinelSecret+`",`),
			setup:   func(m *listenMocks) {},
		},
		{
			name:    "conversations post unmarshal failure",
			request: newReq("/v1/conversations", sock.RequestMethodPost, `{"detail":"`+sentinelSecret+`",`),
			setup:   func(m *listenMocks) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			hook := logtest.Capture(t)

			h, m := newErrorPathHandler(mc)
			tt.setup(m)

			if _, err := h.processRequest(tt.request); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			logtest.AssertNoLeak(t, hook, sentinelSecret, sentinelToken)
		})
	}
}
