package accounthandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
	"monorepo/bin-conversation-manager/pkg/whatsapphandler"
)

func Test_NewAccountHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockLine := linehandler.NewMockLineHandler(mc)
	mockWhatsApp := whatsapphandler.NewMockWhatsAppHandler(mc)

	res, ok := NewAccountHandler(mockDB, mockReq, mockNotify, mockLine, mockWhatsApp).(*accountHandler)
	if !ok {
		t.Fatalf("NewAccountHandler() did not return an *accountHandler")
	}

	if res.db != mockDB || res.reqHandler != mockReq || res.notifyHandler != mockNotify || res.lineHandler != mockLine || res.whatsappHandler != mockWhatsApp {
		t.Errorf("NewAccountHandler() did not keep the given handlers")
	}
	if res.utilHandler == nil {
		t.Errorf("NewAccountHandler() did not create the util handler")
	}
}

func Test_setup_whatsApp(t *testing.T) {
	errSetup := errors.New("provider setup failed")

	tests := []struct {
		name string

		setupErr error

		expectErr bool
	}{
		{name: "the provider setup succeeds"},
		{name: "the provider failure is returned with the context", setupErr: errSetup, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockWhatsApp := whatsapphandler.NewMockWhatsAppHandler(mc)
			// the line handler is a strict mock without calls: only the whatsapp provider is used
			h := accountHandler{
				lineHandler:     linehandler.NewMockLineHandler(mc),
				whatsappHandler: mockWhatsApp,
			}

			ctx := context.Background()
			ac := &account.Account{Type: account.TypeWhatsApp}
			mockWhatsApp.EXPECT().Setup(ctx, ac).Return(tt.setupErr)

			err := h.setup(ctx, ac)
			if !tt.expectErr {
				if err != nil {
					t.Errorf("setup() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, errSetup) || !strings.Contains(err.Error(), "could not setup the account") {
				t.Errorf("setup() error = %v, want the provider error wrapped with the context", err)
			}
		})
	}
}

func Test_setup_unsupportedType(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no provider handler is called for an unknown type
	h := accountHandler{
		lineHandler:     linehandler.NewMockLineHandler(mc),
		whatsappHandler: whatsapphandler.NewMockWhatsAppHandler(mc),
	}

	err := h.setup(context.Background(), &account.Account{Type: account.Type("unknown")})
	if err == nil || !strings.Contains(err.Error(), "unsupported account type") {
		t.Errorf("setup() error = %v, want the unsupported type error", err)
	}
}

func Test_teardown_providers(t *testing.T) {
	tests := []struct {
		name string

		accountType account.Type
		expectLine  bool
	}{
		{name: "line calls the provider", accountType: account.TypeLine, expectLine: true},
		{name: "sms has nothing to tear down", accountType: account.TypeSMS},
		{name: "whatsapp has nothing to tear down", accountType: account.TypeWhatsApp},
		{name: "an unknown type has nothing to tear down", accountType: account.Type("unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockLine := linehandler.NewMockLineHandler(mc)
			h := accountHandler{
				lineHandler:     mockLine,
				whatsappHandler: whatsapphandler.NewMockWhatsAppHandler(mc),
			}

			ctx := context.Background()
			ac := &account.Account{Type: tt.accountType}
			if tt.expectLine {
				// a provider failure is only logged
				mockLine.EXPECT().Teardown(ctx, ac).Return(errors.New("provider teardown failed")).Times(1)
			}

			h.teardown(ctx, ac)
		})
	}
}
