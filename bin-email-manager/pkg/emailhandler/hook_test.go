package emailhandler

import (
	"context"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/pkg/dbhandler"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_Hook(t *testing.T) {

	tests := []struct {
		name string

		uri  string
		data []byte

		expectIDs      []uuid.UUID
		expectStatuses []email.Status
	}{
		{
			name: "normal",

			uri:  "hook.voipbin.net/v1.0/emails/sendgrid",
			data: []byte(`[{"email":"pchero21@gmail.com","event":"delivered","ip":"149.72.123.24","voipbin_message_id":"b90a8049-b235-443e-b17e-07a5d2e59a79","response":"250 2.0.0 OK  1741882953 af79cd13be357-7c573d1a920si196001785a.586 - gsmtp","sg_event_id":"ZGVsaXZlcmVkLTAtNTEyMzAwODMtc0NtMHBmRFBSanktSHloSEJfNVR1dy0w","sg_message_id":"sCm0pfDPRjy-HyhHB_5Tuw.recvd-69cbc48b94-n2vpp-1-67D30649-1.0","smtp-id":"<sCm0pfDPRjy-HyhHB_5Tuw@geopod-ismtpd-8>","timestamp":1741882953,"tls":1},{"email":"pchero21@gmail.com","event":"processed","voipbin_message_id":"b90a8049-b235-443e-b17e-07a5d2e59a79","send_at":0,"sg_event_id":"cHJvY2Vzc2VkLTUxMjMwMDgzLXNDbTBwZkRQUmp5LUh5aEhCXzVUdXctMA","sg_message_id":"sCm0pfDPRjy-HyhHB_5Tuw.recvd-69cbc48b94-n2vpp-1-67D30649-1.0","smtp-id":"<sCm0pfDPRjy-HyhHB_5Tuw@geopod-ismtpd-8>","timestamp":1741882953}]`),

			expectIDs: []uuid.UUID{
				uuid.FromStringOrNil("b90a8049-b235-443e-b17e-07a5d2e59a79"),
				uuid.FromStringOrNil("b90a8049-b235-443e-b17e-07a5d2e59a79"),
			},
			expectStatuses: []email.Status{
				email.StatusProcessed,
				email.StatusDelivered,
			},
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
			ctx := context.Background()

			for i := range tt.expectIDs {
				mockDB.EXPECT().EmailUpdateStatus(ctx, tt.expectIDs[i], tt.expectStatuses[i].Return(nil)
				mockDB.EXPECT().EmailGet(ctx, tt.expectIDs[i].Return(&email.Email{}, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), email.EventTypeUpdated, gomock.Any())
			}

			if err := h.Hook(ctx, tt.uri, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_hookSendgrid(t *testing.T) {

	tests := []struct {
		name string

		data []byte

		expectIDs      []uuid.UUID
		expectStatuses []email.Status
	}{
		{
			name: "normal",

			data: []byte(`[{"email":"pchero21@gmail.com","event":"delivered","ip":"149.72.123.24","voipbin_message_id":"b90a8049-b235-443e-b17e-07a5d2e59a79","response":"250 2.0.0 OK  1741882953 af79cd13be357-7c573d1a920si196001785a.586 - gsmtp","sg_event_id":"ZGVsaXZlcmVkLTAtNTEyMzAwODMtc0NtMHBmRFBSanktSHloSEJfNVR1dy0w","sg_message_id":"sCm0pfDPRjy-HyhHB_5Tuw.recvd-69cbc48b94-n2vpp-1-67D30649-1.0","smtp-id":"<sCm0pfDPRjy-HyhHB_5Tuw@geopod-ismtpd-8>","timestamp":1741882953,"tls":1},{"email":"pchero21@gmail.com","event":"processed","voipbin_message_id":"b90a8049-b235-443e-b17e-07a5d2e59a79","send_at":0,"sg_event_id":"cHJvY2Vzc2VkLTUxMjMwMDgzLXNDbTBwZkRQUmp5LUh5aEhCXzVUdXctMA","sg_message_id":"sCm0pfDPRjy-HyhHB_5Tuw.recvd-69cbc48b94-n2vpp-1-67D30649-1.0","smtp-id":"<sCm0pfDPRjy-HyhHB_5Tuw@geopod-ismtpd-8>","timestamp":1741882953}]`),

			expectIDs: []uuid.UUID{
				uuid.FromStringOrNil("b90a8049-b235-443e-b17e-07a5d2e59a79"),
				uuid.FromStringOrNil("b90a8049-b235-443e-b17e-07a5d2e59a79"),
			},
			expectStatuses: []email.Status{
				email.StatusProcessed,
				email.StatusDelivered,
			},
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
			ctx := context.Background()

			for i := range tt.expectIDs {
				mockDB.EXPECT().EmailUpdateStatus(ctx, tt.expectIDs[i], tt.expectStatuses[i].Return(nil)
				mockDB.EXPECT().EmailGet(ctx, tt.expectIDs[i].Return(&email.Email{}, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), email.EventTypeUpdated, gomock.Any())
			}

			if err := h.hookSendgrid(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
