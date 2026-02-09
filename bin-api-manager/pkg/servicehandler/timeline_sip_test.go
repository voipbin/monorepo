package servicehandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcall "monorepo/bin-call-manager/models/call"
	cmchannel "monorepo/bin-call-manager/models/channel"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	tmsipmessage "monorepo/bin-timeline-manager/models/sipmessage"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_TimelineSIPAnalysisGet(t *testing.T) {

	callID := uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a")
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")

	now := time.Now()
	tmCreate := now.Add(-10 * time.Minute)
	tmHangup := now.Add(-5 * time.Minute)

	call := &cmcall.Call{
		Identity: commonidentity.Identity{
			ID:         callID,
			CustomerID: customerID,
		},
		ChannelID: "channel-123",
		TMCreate:  &tmCreate,
		TMHangup:  &tmHangup,
	}

	channel := &cmchannel.Channel{
		ID:        "channel-123",
		SIPCallID: "sip-call-id-abc123",
	}

	tests := []struct {
		name string

		agent  *amagent.Agent
		callID uuid.UUID

		responseCall     *cmcall.Call
		responseChannel  *cmchannel.Channel
		responseAnalysis *tmsipmessage.SIPAnalysisResponse

		expectRes *tmsipmessage.SIPAnalysisResponse
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			callID: callID,

			responseCall:    call,
			responseChannel: channel,
			responseAnalysis: &tmsipmessage.SIPAnalysisResponse{
				SIPMessages: []*tmsipmessage.SIPMessage{
					{
						Method: "INVITE",
						SrcIP:  "10.0.0.1",
					},
				},
			},

			expectRes: &tmsipmessage.SIPAnalysisResponse{
				SIPMessages: []*tmsipmessage.SIPMessage{
					{
						Method: "INVITE",
						SrcIP:  "10.0.0.1",
					},
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.responseCall, nil)
			mockReq.EXPECT().CallV1ChannelGet(ctx, "channel-123").Return(tt.responseChannel, nil)
			mockReq.EXPECT().TimelineV1SIPAnalysisGet(ctx, tt.callID, "sip-call-id-abc123", tmCreate.Format(time.RFC3339), tmHangup.Format(time.RFC3339)).Return(tt.responseAnalysis, nil)

			res, err := h.TimelineSIPAnalysisGet(ctx, tt.agent, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TimelineSIPPcapGet(t *testing.T) {

	callID := uuid.FromStringOrNil("fe003a08-8f36-11ed-a01a-efb53befe93a")
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")

	now := time.Now()
	tmCreate := now.Add(-10 * time.Minute)
	tmHangup := now.Add(-5 * time.Minute)

	call := &cmcall.Call{
		Identity: commonidentity.Identity{
			ID:         callID,
			CustomerID: customerID,
		},
		ChannelID: "channel-123",
		TMCreate:  &tmCreate,
		TMHangup:  &tmHangup,
	}

	channel := &cmchannel.Channel{
		ID:        "channel-123",
		SIPCallID: "sip-call-id-abc123",
	}

	tests := []struct {
		name string

		agent  *amagent.Agent
		callID uuid.UUID

		responseCall    *cmcall.Call
		responseChannel *cmchannel.Channel
		responsePcap    []byte

		expectRes []byte
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			callID: callID,

			responseCall:    call,
			responseChannel: channel,
			responsePcap:    []byte("pcap-binary-data"),

			expectRes: []byte("pcap-binary-data"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.responseCall, nil)
			mockReq.EXPECT().CallV1ChannelGet(ctx, "channel-123").Return(tt.responseChannel, nil)
			mockReq.EXPECT().TimelineV1SIPPcapGet(ctx, tt.callID, "sip-call-id-abc123", tmCreate.Format(time.RFC3339), tmHangup.Format(time.RFC3339)).Return(tt.responsePcap, nil)

			res, err := h.TimelineSIPPcapGet(ctx, tt.agent, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
