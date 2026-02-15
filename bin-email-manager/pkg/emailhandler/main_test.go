package emailhandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-email-manager/pkg/dbhandler"

	"go.uber.org/mock/gomock"
)

func TestNewEmailHandler(t *testing.T) {
	tests := []struct {
		name           string
		sendgridAPIKey string
		mailgunAPIKey  string
	}{
		{
			name:           "creates_new_email_handler",
			sendgridAPIKey: "test-sendgrid-key",
			mailgunAPIKey:  "test-mailgun-key",
		},
		{
			name:           "creates_with_empty_keys",
			sendgridAPIKey: "",
			mailgunAPIKey:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := NewEmailHandler(mockDB, mockReq, mockNotify, tt.sendgridAPIKey, tt.mailgunAPIKey)

			if h == nil {
				t.Errorf("Expected non-nil handler")
			}
		})
	}
}
