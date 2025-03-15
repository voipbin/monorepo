package emailhandler

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-email-manager/pkg/dbhandler"
	"testing"

	"go.uber.org/mock/gomock"
)

func Test_validateEmailAddress(t *testing.T) {

	tests := []struct {
		name string

		target commonaddress.Address

		expectedRes bool
	}{
		{
			name: "normal",

			target: commonaddress.Address{
				Type:   commonaddress.TypeEmail,
				Target: "test@voipbin.net",
			},

			expectedRes: true,
		},
		{
			name: "target address is ok, but the type is not email",

			target: commonaddress.Address{
				Type:   commonaddress.TypeAgent,
				Target: "test@voipbin.net",
			},

			expectedRes: false,
		},
		{
			name: "type is email but the target is not valid email format",

			target: commonaddress.Address{
				Type:   commonaddress.TypeEmail,
				Target: "wrong email format",
			},

			expectedRes: false,
		},
		{
			name: "type is email but the target is empty",

			target: commonaddress.Address{
				Type:   commonaddress.TypeEmail,
				Target: "",
			},

			expectedRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSendgrid := NewMockEngineSendgrid(mc)

			h := &emailHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,

				engineSendgrid: mockSendgrid,
			}

			res := h.validateEmailAddress(tt.target)
			if res != tt.expectedRes {
				t.Errorf("Wrong match. expected: %v, got: %v", tt.expectedRes, res)
			}
		})
	}
}
