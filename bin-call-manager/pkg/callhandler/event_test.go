package callhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/outboundconfighandler"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"
	smcontainer "monorepo/bin-sentinel-manager/models/container"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_EventCUCustomerDeleted(t *testing.T) {

	customerID := uuid.FromStringOrNil("8c0daf80-f0c3-11ee-9ed5-6b65132a6fc3")
	configID := uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001")

	tests := []struct {
		name string

		customer       *cmcustomer.Customer
		responseCalls  []*call.Call
		responseConfig *outboundconfig.OutboundConfig

		expectFilter map[string]string
	}{
		{
			name: "normal",

			customer: &cmcustomer.Customer{
				ID: customerID,
			},
			responseCalls: []*call.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8c70ee42-f0c3-11ee-b8d2-b3b3892bc551"),
					},
					Status:   call.StatusHangup,
					TMDelete: nil,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8c9af3f4-f0c3-11ee-9351-cfa1330e7d25"),
					},
					Status:   call.StatusHangup,
					TMDelete: nil,
				},
			},
			responseConfig: &outboundconfig.OutboundConfig{
				ID:         configID,
				CustomerID: customerID,
			},

			expectFilter: map[string]string{
				"customer_id": "8c0daf80-f0c3-11ee-9ed5-6b65132a6fc3",
				"deleted":     "false",
			},
		},
		{
			name: "no outbound config",

			customer: &cmcustomer.Customer{
				ID: customerID,
			},
			responseCalls:  []*call.Call{},
			responseConfig: nil,
		},
	}

	// Also verify the error path: GetByCustomerID returns an error → handler logs warn and returns nil (non-fatal)
	t.Run("outbound config get error is non-fatal", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockDB := dbhandler.NewMockDBHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		mockUtil := utilhandler.NewMockUtilHandler(mc)
		mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)

		h := &callHandler{
			reqHandler:            mockReq,
			db:                    mockDB,
			notifyHandler:         mockNotify,
			utilHandler:           mockUtil,
			outboundConfigHandler: mockOutboundConfig,
		}
		ctx := context.Background()

		customer := &cmcustomer.Customer{ID: customerID}

		mockDB.EXPECT().CallList(ctx, uint64(1000), "", gomock.Any()).Return([]*call.Call{}, nil)
		// GetByCustomerID returns an error — handler must log Warn and return nil (not propagate the error)
		mockOutboundConfig.EXPECT().GetByCustomerID(ctx, customer.ID).Return(nil, fmt.Errorf("db unavailable"))

		if err := h.EventCUCustomerDeleted(ctx, customer); err != nil {
			t.Errorf("EventCUCustomerDeleted must return nil even when OutboundConfig get fails, got: %v", err)
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)

			h := &callHandler{
				reqHandler:            mockReq,
				db:                    mockDB,
				notifyHandler:         mockNotify,
				utilHandler:           mockUtil,
				outboundConfigHandler: mockOutboundConfig,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallList(ctx, uint64(1000), "", gomock.Any()).Return(tt.responseCalls, nil)

			// delete each call
			for _, c := range tt.responseCalls {
				mockDB.EXPECT().CallGet(ctx, c.ID).Return(c, nil)
				mockDB.EXPECT().CallDelete(ctx, c.ID).Return(nil)
				mockDB.EXPECT().CallGet(ctx, c.ID).Return(c, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, c.CustomerID, call.EventTypeCallDeleted, c)
			}

			// delete outbound config
			mockOutboundConfig.EXPECT().GetByCustomerID(ctx, tt.customer.ID).Return(tt.responseConfig, nil)
			if tt.responseConfig != nil {
				mockOutboundConfig.EXPECT().Delete(ctx, tt.responseConfig.ID).Return(tt.responseConfig, nil)
			}

			if err := h.EventCUCustomerDeleted(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventCUCustomerFrozen(t *testing.T) {

	tests := []struct {
		name string

		customer      *cmcustomer.Customer
		responseCalls []*call.Call
	}{
		{
			name: "normal",

			customer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("8c0daf80-f0c3-11ee-9ed5-6b65132a6fc3"),
			},
			responseCalls: []*call.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8c70ee42-f0c3-11ee-b8d2-b3b3892bc551"),
					},
					Status: call.StatusTerminating,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8c9af3f4-f0c3-11ee-9351-cfa1330e7d25"),
					},
					Status: call.StatusTerminating,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &callHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallList(ctx, uint64(1000), "", gomock.Any()).Return(tt.responseCalls, nil)

			// hangup each call - HangingUp calls Get, then returns early because status is already terminating
			for _, c := range tt.responseCalls {
				mockDB.EXPECT().CallGet(ctx, c.ID).Return(c, nil)
			}

			if err := h.EventCUCustomerFrozen(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

// Test_EventSMContainerDied covers the sentinel-manager consumer path end to end.
//
// This is NET-NEW coverage: the predecessor, EventSMPodDeleted, had no test at all. The three
// branches that matter are the happy path, the service filter, and -- new in VOIP-1418 -- the
// empty-asterisk-id guard, which did not exist before and is the only thing standing between an
// unresolved sentinel event and a RecoveryStart run against an empty asterisk-id.
func Test_EventSMContainerDied(t *testing.T) {
	startTime := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		event *smcontainer.Event

		expectRecovery   bool
		expectAsteriskID string
	}{
		{
			name: "normal",

			event: &smcontainer.Event{
				ContainerName: "voip-asterisk-call-docker-1",
				Service:       smcontainer.ServiceAsteriskCall,
				AsteriskID:    "3e:50:6b:43:bb:32",
			},

			expectRecovery:   true,
			expectAsteriskID: "3e:50:6b:43:bb:32",
		},
		{
			name: "another replica",

			event: &smcontainer.Event{
				ContainerName: "voip-asterisk-call-docker-2",
				Service:       smcontainer.ServiceAsteriskCall,
				AsteriskID:    "72:ce:24:e6:51:2f",
			},

			expectRecovery:   true,
			expectAsteriskID: "72:ce:24:e6:51:2f",
		},
		{
			name: "conference container is filtered out",

			event: &smcontainer.Event{
				ContainerName: "voip-asterisk-conference-docker-1",
				Service:       smcontainer.ServiceAsteriskConference,
				AsteriskID:    "3e:50:6b:43:bb:32",
			},

			expectRecovery: false,
		},
		{
			name: "registrar container is filtered out",

			event: &smcontainer.Event{
				ContainerName: "voip-asterisk-registrar-docker-2",
				Service:       smcontainer.ServiceAsteriskRegistrar,
				AsteriskID:    "3e:50:6b:43:bb:32",
			},

			expectRecovery: false,
		},
		{
			name: "empty service is filtered out",

			event: &smcontainer.Event{
				ContainerName: "voip-asterisk-call-docker-1",
				Service:       "",
				AsteriskID:    "3e:50:6b:43:bb:32",
			},

			expectRecovery: false,
		},
		{
			// `json.Unmarshal` of a literal `null` body into a **Event succeeds and leaves the
			// pointer nil. Without the guard this panics the whole subscribe loop on the first
			// field read.
			name: "nil event is skipped without panicking",

			event: nil,

			expectRecovery: false,
		},
		{
			name: "unresolved asterisk id skips the recovery",

			event: &smcontainer.Event{
				ContainerName: "voip-asterisk-call-docker-1",
				Service:       smcontainer.ServiceAsteriskCall,
				AsteriskID:    "",
			},

			expectRecovery: false,
		},
		{
			name: "unresolved asterisk id on a filtered service skips the recovery too",

			event: &smcontainer.Event{
				ContainerName: "voip-asterisk-registrar-docker-1",
				Service:       smcontainer.ServiceAsteriskRegistrar,
				AsteriskID:    "",
			},

			expectRecovery: false,
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
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			if tt.expectRecovery {
				// RecoveryStart is a method on this same handler, so the strictest observable
				// assertion is the channel lookup it performs with the asterisk-id.
				mockUtil.EXPECT().TimeNowAdd(-(time.Hour * 24)).Return(&startTime)
				mockUtil.EXPECT().TimeNow().Return(&endTime)
				mockChannel.EXPECT().
					GetChannelsForRecovery(ctx, tt.expectAsteriskID, channel.TypeCall, &startTime, &endTime, defaultRecoveryChannelLimit).
					Return([]*channel.Channel{}, nil)
			}
			// when no recovery is expected, the strict mock controller itself is the assertion:
			// any GetChannelsForRecovery call would fail the test.

			if err := h.EventSMContainerDied(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

// Test_EventSMContainerDied_recoveryError pins the error path: a failing recovery is wrapped and
// returned, not swallowed.
func Test_EventSMContainerDied_recoveryError(t *testing.T) {
	startTime := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockChannel := channelhandler.NewMockChannelHandler(mc)

	h := &callHandler{
		utilHandler:    mockUtil,
		reqHandler:     requesthandler.NewMockRequestHandler(mc),
		db:             dbhandler.NewMockDBHandler(mc),
		notifyHandler:  notifyhandler.NewMockNotifyHandler(mc),
		channelHandler: mockChannel,
	}

	ctx := context.Background()

	mockUtil.EXPECT().TimeNowAdd(-(time.Hour * 24)).Return(&startTime)
	mockUtil.EXPECT().TimeNow().Return(&endTime)
	mockChannel.EXPECT().
		GetChannelsForRecovery(ctx, "3e:50:6b:43:bb:32", channel.TypeCall, &startTime, &endTime, defaultRecoveryChannelLimit).
		Return(nil, fmt.Errorf("could not query the channels"))

	event := &smcontainer.Event{
		ContainerName: "voip-asterisk-call-docker-1",
		Service:       smcontainer.ServiceAsteriskCall,
		AsteriskID:    "3e:50:6b:43:bb:32",
	}

	if err := h.EventSMContainerDied(ctx, event); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}
