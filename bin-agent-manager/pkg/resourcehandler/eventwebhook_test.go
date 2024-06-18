package resourcehandler

import (
	"context"
	"monorepo/bin-agent-manager/models/resource"
	"monorepo/bin-agent-manager/pkg/agenthandler"
	"monorepo/bin-agent-manager/pkg/dbhandler"
	cmcall "monorepo/bin-call-manager/models/call"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
)

// func Test_eventWebhookCallCreated(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		call *cmcall.WebhookMessage

// 		responseAgents []*agent.Agent
// 		expectAddr     commonaddress.Address
// 	}{
// 		{
// 			name: "normal incoming",

// 			call: &cmcall.WebhookMessage{
// 				ID:        uuid.FromStringOrNil("77d87e1e-2a57-11ef-aa71-dbaadea01cc5"),
// 				Direction: cmcall.DirectionIncoming,
// 				Source: commonaddress.Address{
// 					Type:   commonaddress.TypeTel,
// 					Target: "+1123456",
// 				},
// 				Destination: commonaddress.Address{
// 					Type:   commonaddress.TypeTel,
// 					Target: "+12345689",
// 				},
// 			},

// 			responseAgents: []*agent.Agent{
// 				{
// 					ID: uuid.FromStringOrNil("73deea7c-2a58-11ef-967c-4f58981813df"),
// 				},
// 				{
// 					ID: uuid.FromStringOrNil("745ee1a0-2a58-11ef-a47e-3f9496dda121"),
// 				},
// 			},

// 			expectAddr: commonaddress.Address{
// 				Type:   commonaddress.TypeTel,
// 				Target: "+1123456",
// 			},
// 		},
// 		{
// 			name: "normal outgoing",

// 			call: &cmcall.WebhookMessage{
// 				ID:        uuid.FromStringOrNil("cac3459a-2a58-11ef-a3cd-9fea9818f587"),
// 				Direction: cmcall.DirectionOutgoing,
// 				Source: commonaddress.Address{
// 					Type:   commonaddress.TypeTel,
// 					Target: "+1123456",
// 				},
// 				Destination: commonaddress.Address{
// 					Type:   commonaddress.TypeTel,
// 					Target: "+12345689",
// 				},
// 			},

// 			responseAgents: []*agent.Agent{
// 				{
// 					ID: uuid.FromStringOrNil("cb291104-2a58-11ef-acf3-2b9c33f111b5"),
// 				},
// 				{
// 					ID: uuid.FromStringOrNil("cb4a69a8-2a58-11ef-af5d-d3f987e61c65"),
// 				},
// 			},

// 			expectAddr: commonaddress.Address{
// 				Type:   commonaddress.TypeTel,
// 				Target: "+12345689",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockUtil := utilhandler.NewMockUtilHandler(mc)
// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockAgent := agenthandler.NewMockAgentHandler(mc)

// 			h := &resourceHandler{
// 				utilHandler:   mockUtil,
// 				reqHandler:    mockReq,
// 				db:            mockDB,
// 				notifyHandler: mockNotify,
// 				agentHandler:  mockAgent,
// 			}
// 			ctx := context.Background()

// 			mockAgent.EXPECT().GetsByCustomerIDAndAddress(ctx, tt.call.CustomerID, tt.expectAddr).Return(tt.responseAgents, nil)
// 			for range tt.responseAgents {
// 				mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
// 				mockDB.EXPECT().ResourceCreate(ctx, gomock.Any()).Return(nil)
// 				mockDB.EXPECT().ResourceGet(ctx, gomock.Any()).Return(&resource.Resource{}, nil)
// 				mockNotify.EXPECT().PublishEvent(ctx, resource.EventTypeResourceCreated, gomock.Any())
// 			}

// 			if err := h.eventWebhookCallCreated(ctx, tt.call); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 		})
// 	}
// }

func Test_eventWebhookCallUpdated(t *testing.T) {

	tests := []struct {
		name string

		call *cmcall.WebhookMessage

		responseResources []*resource.Resource
		expectFilters     map[string]string
	}{
		{
			name: "normal incoming",

			call: &cmcall.WebhookMessage{
				ID:         uuid.FromStringOrNil("0a3553ee-2a59-11ef-96c6-43f63d37f4f9"),
				CustomerID: uuid.FromStringOrNil("52fe1a2a-2a59-11ef-b3cd-3b870a5dcf84"),
			},

			responseResources: []*resource.Resource{
				{
					ID: uuid.FromStringOrNil("0af8c536-2a59-11ef-80e9-879c18feda0f"),
				},
				{
					ID: uuid.FromStringOrNil("0b195fe4-2a59-11ef-bee8-7bdae6508aa8"),
				},
			},
			expectFilters: map[string]string{
				"customer_id":    "52fe1a2a-2a59-11ef-b3cd-3b870a5dcf84",
				"reference_type": string(resource.ReferenceTypeCall),
				"reference_id":   "0a3553ee-2a59-11ef-96c6-43f63d37f4f9",
				"deleted":        "false",
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
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := &resourceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				agentHandler:  mockAgent,
			}
			ctx := context.Background()

			mockDB.EXPECT().ResourceGets(ctx, uint64(100), "", tt.expectFilters).Return(tt.responseResources, nil)
			for _, r := range tt.responseResources {
				mockDB.EXPECT().ResourceSetData(ctx, r.ID, tt.call).Return(nil)
				mockDB.EXPECT().ResourceGet(ctx, r.ID).Return(&resource.Resource{}, nil)
				mockNotify.EXPECT().PublishEvent(ctx, resource.EventTypeResourceUpdated, gomock.Any())
			}

			if err := h.eventWebhookCallUpdated(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_eventWebhookCallDeleted(t *testing.T) {

	tests := []struct {
		name string

		call *cmcall.WebhookMessage

		responseResources []*resource.Resource
		expectFilters     map[string]string
	}{
		{
			name: "normal incoming",

			call: &cmcall.WebhookMessage{
				ID:         uuid.FromStringOrNil("96ba8208-2a63-11ef-8b4c-6bf84644b8b6"),
				CustomerID: uuid.FromStringOrNil("973b1d50-2a63-11ef-a0e9-bba40d0c2b0b"),
			},

			responseResources: []*resource.Resource{
				{
					ID: uuid.FromStringOrNil("976bd40e-2a63-11ef-8d03-a3029849d597"),
				},
				{
					ID: uuid.FromStringOrNil("979f3920-2a63-11ef-9cbf-cfe94406e39b"),
				},
			},
			expectFilters: map[string]string{
				"customer_id":    "973b1d50-2a63-11ef-a0e9-bba40d0c2b0b",
				"reference_type": string(resource.ReferenceTypeCall),
				"reference_id":   "96ba8208-2a63-11ef-8b4c-6bf84644b8b6",
				"deleted":        "false",
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
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := &resourceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				agentHandler:  mockAgent,
			}
			ctx := context.Background()

			mockDB.EXPECT().ResourceGets(ctx, uint64(1000), "", tt.expectFilters).Return(tt.responseResources, nil)
			for _, r := range tt.responseResources {
				mockDB.EXPECT().ResourceDelete(ctx, r.ID).Return(nil)
				mockDB.EXPECT().ResourceGet(ctx, r.ID).Return(&resource.Resource{}, nil)
				mockNotify.EXPECT().PublishEvent(ctx, resource.EventTypeResourceDeleted, gomock.Any())
			}

			if err := h.eventWebhookCallDeleted(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
