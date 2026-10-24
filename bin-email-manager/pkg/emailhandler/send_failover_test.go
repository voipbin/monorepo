package emailhandler

import (
	"context"
	"fmt"
	"testing"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_Send_failover(t *testing.T) {
	e := &email.Email{
		Identity: identity.Identity{ID: uuid.FromStringOrNil("5e8f0f0a-6666-4666-8666-000000000001")},
	}

	tests := []struct {
		name string

		setup func(ctx context.Context, sg *MockEngineSendgrid, mg *MockEngineMailgun, db *dbhandler.MockDBHandler)
	}{
		{
			name: "sendgrid_fails_mailgun_succeeds",
			setup: func(ctx context.Context, sg *MockEngineSendgrid, mg *MockEngineMailgun, db *dbhandler.MockDBHandler) {
				sg.EXPECT().Send(ctx, e).Return("", fmt.Errorf("sendgrid down"))
				mg.EXPECT().Send(ctx, e).Return("mailgun-ref", nil)
				db.EXPECT().EmailUpdateProviderReferenceID(ctx, e.ID, "mailgun-ref").Return(nil)
			},
		},
		{
			name: "both_providers_fail_leaves_email_untouched",
			setup: func(ctx context.Context, sg *MockEngineSendgrid, mg *MockEngineMailgun, db *dbhandler.MockDBHandler) {
				sg.EXPECT().Send(ctx, e).Return("", fmt.Errorf("sendgrid down"))
				mg.EXPECT().Send(ctx, e).Return("", fmt.Errorf("mailgun down"))
			},
		},
		{
			name: "reference_update_failure_does_not_retry_next_provider",
			setup: func(ctx context.Context, sg *MockEngineSendgrid, mg *MockEngineMailgun, db *dbhandler.MockDBHandler) {
				sg.EXPECT().Send(ctx, e).Return("sendgrid-ref", nil)
				db.EXPECT().EmailUpdateProviderReferenceID(ctx, e.ID, "sendgrid-ref").Return(fmt.Errorf("db down"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockSendgrid := NewMockEngineSendgrid(mc)
			mockMailgun := NewMockEngineMailgun(mc)

			h := &emailHandler{
				db:             mockDB,
				reqHandler:     requesthandler.NewMockRequestHandler(mc),
				notifyHandler:  notifyhandler.NewMockNotifyHandler(mc),
				utilHandler:    utilhandler.NewMockUtilHandler(mc),
				engineSendgrid: mockSendgrid,
				engineMailgun:  mockMailgun,
			}
			ctx := context.Background()

			tt.setup(ctx, mockSendgrid, mockMailgun, mockDB)

			// gomock fails the test on any call that was not declared above
			h.Send(ctx, e)
		})
	}
}

func Test_hookSendgrid_skipsBadEvents(t *testing.T) {
	good := uuid.FromStringOrNil("5e8f0f0a-7777-4777-8777-000000000001")
	failing := uuid.FromStringOrNil("5e8f0f0a-7777-4777-8777-000000000002")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &emailHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	// events are processed in reverse order: the failing one first, then the one
	// without a message id (skipped), then the good one.
	data := []byte(`[
		{"event":"delivered","voipbin_message_id":"` + good.String() + `"},
		{"event":"processed"},
		{"event":"open","voipbin_message_id":"` + failing.String() + `"}
	]`)

	gomock.InOrder(
		mockDB.EXPECT().EmailUpdateStatus(ctx, failing, email.StatusOpen).Return(fmt.Errorf("db down")),
		mockDB.EXPECT().EmailUpdateStatus(ctx, good, email.StatusDelivered).Return(nil),
		mockDB.EXPECT().EmailGet(ctx, good).Return(&email.Email{Identity: identity.Identity{ID: good}}, nil),
		mockNotify.EXPECT().PublishWebhookEvent(ctx, uuid.Nil, email.EventTypeUpdated, gomock.Any()),
	)

	if err := h.hookSendgrid(ctx, data); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}
