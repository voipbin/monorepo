package agenthandler

// func Test_webhookCallCreated(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		call *cmcall.Call

// 		responseAgents []*agent.Agent
// 		expectAddr     commonaddress.Address
// 	}{
// 		{
// 			name: "normal incoming",

// 			call: &cmcall.Call{
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

// 			call: &cmcall.Call{
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

// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockResource := resourcehandler.NewMockResourceHandler(mc)

// 			h := &agentHandler{
// 				reqHandler:      mockReq,
// 				db:              mockDB,
// 				notifyHandler:   mockNotify,
// 				resourceHandler: mockResource,
// 			}
// 			ctx := context.Background()

// 			mockDB.EXPECT().AgentGetsByCustomerIDAndAddress(ctx, tt.call.CustomerID, tt.expectAddr).Return(tt.responseAgents, nil)
// 			for _, a := range tt.responseAgents {
// 				mockResource.EXPECT().Create(ctx, tt.call.CustomerID, a.ID, resource.ReferenceTypeCall, tt.call.ID, tt.call).Return(&resource.Resource{}, nil)
// 			}

// 			if err := h.webhookCallCreated(ctx, tt.call); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 		})
// 	}
// }

// func Test_webhookCallUpdated(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		call *cmcall.Call

// 		responseResources []*resource.Resource
// 		expectFilters     map[string]string
// 	}{
// 		{
// 			name: "normal incoming",

// 			call: &cmcall.Call{
// 				ID:         uuid.FromStringOrNil("0a3553ee-2a59-11ef-96c6-43f63d37f4f9"),
// 				CustomerID: uuid.FromStringOrNil("52fe1a2a-2a59-11ef-b3cd-3b870a5dcf84"),
// 				Direction:  cmcall.DirectionIncoming,
// 				Source: commonaddress.Address{
// 					Type:   commonaddress.TypeTel,
// 					Target: "+1123456",
// 				},
// 				Destination: commonaddress.Address{
// 					Type:   commonaddress.TypeTel,
// 					Target: "+12345689",
// 				},
// 			},

// 			responseResources: []*resource.Resource{
// 				{
// 					ID: uuid.FromStringOrNil("0af8c536-2a59-11ef-80e9-879c18feda0f"),
// 				},
// 				{
// 					ID: uuid.FromStringOrNil("0b195fe4-2a59-11ef-bee8-7bdae6508aa8"),
// 				},
// 			},
// 			expectFilters: map[string]string{
// 				"customer_id":    "52fe1a2a-2a59-11ef-b3cd-3b870a5dcf84",
// 				"reference_type": string(resource.ReferenceTypeCall),
// 				"reference_id":   "0a3553ee-2a59-11ef-96c6-43f63d37f4f9",
// 				"deleted":        "false",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockResource := resourcehandler.NewMockResourceHandler(mc)

// 			h := &agentHandler{
// 				reqHandler:      mockReq,
// 				db:              mockDB,
// 				notifyHandler:   mockNotify,
// 				resourceHandler: mockResource,
// 			}
// 			ctx := context.Background()

// 			mockResource.EXPECT().Gets(ctx, uint64(100), "", tt.expectFilters).Return(tt.responseResources, nil)
// 			for _, r := range tt.responseResources {
// 				mockResource.EXPECT().UpdateData(ctx, r.ID, tt.call).Return(&resource.Resource{}, nil)
// 			}

// 			if err := h.webhookCallUpdated(ctx, tt.call); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 		})
// 	}
// }
