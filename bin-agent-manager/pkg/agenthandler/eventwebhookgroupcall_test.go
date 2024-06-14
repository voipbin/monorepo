package agenthandler

// func Test_webhookGroupcallCreated(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		groupcall *cmgroupcall.Groupcall

// 		responseAgents [][]*agent.Agent
// 		expectAddr     commonaddress.Address
// 	}{
// 		{
// 			name: "normal",

// 			groupcall: &cmgroupcall.Groupcall{
// 				ID: uuid.FromStringOrNil("77d87e1e-2a57-11ef-aa71-dbaadea01cc5"),
// 				Destinations: []commonaddress.Address{
// 					{
// 						Type:   commonaddress.TypeTel,
// 						Target: "+1234567",
// 					},
// 					{
// 						Type:   commonaddress.TypeTel,
// 						Target: "+234567",
// 					},
// 				},
// 			},

// 			responseAgents: [][]*agent.Agent{
// 				{
// 					{
// 						ID: uuid.FromStringOrNil("73deea7c-2a58-11ef-967c-4f58981813df"),
// 					},
// 					{
// 						ID: uuid.FromStringOrNil("745ee1a0-2a58-11ef-a47e-3f9496dda121"),
// 					},
// 				},
// 				{
// 					{
// 						ID: uuid.FromStringOrNil("74623de4-2a5a-11ef-845e-73d29f9d338a"),
// 					},
// 					{
// 						ID: uuid.FromStringOrNil("7496a804-2a5a-11ef-9009-8fd351247222"),
// 					},
// 				},
// 			},

// 			expectAddr: commonaddress.Address{
// 				Type:   commonaddress.TypeTel,
// 				Target: "+1123456",
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

// 			for i, addr := range tt.groupcall.Destinations {
// 				mockDB.EXPECT().AgentGetsByCustomerIDAndAddress(ctx, tt.groupcall.CustomerID, addr).Return(tt.responseAgents[i], nil)
// 				for _, a := range tt.responseAgents[i] {
// 					mockResource.EXPECT().Create(ctx, tt.groupcall.CustomerID, a.ID, resource.ReferenceTypeGroupcall, tt.groupcall.ID, tt.groupcall).Return(&resource.Resource{}, nil)
// 				}
// 			}

// 			if err := h.webhookGroupcallCreated(ctx, tt.groupcall); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 		})
// 	}
// }

// func Test_webhookGroupcallUpdated(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		groupcall *cmgroupcall.Groupcall

// 		responseResources []*resource.Resource
// 		expectFilters     map[string]string
// 	}{
// 		{
// 			name: "normal incoming",

// 			groupcall: &cmgroupcall.Groupcall{
// 				ID:         uuid.FromStringOrNil("15592456-2a5b-11ef-8a97-27d50d25a280"),
// 				CustomerID: uuid.FromStringOrNil("15812ac8-2a5b-11ef-acde-77296c42e687"),
// 			},

// 			responseResources: []*resource.Resource{
// 				{
// 					ID: uuid.FromStringOrNil("15a53a76-2a5b-11ef-9ad4-473d2520acfe"),
// 				},
// 				{
// 					ID: uuid.FromStringOrNil("15cd9a70-2a5b-11ef-97cf-d74756757d0a"),
// 				},
// 			},
// 			expectFilters: map[string]string{
// 				"customer_id":    "15812ac8-2a5b-11ef-acde-77296c42e687",
// 				"reference_type": string(resource.ReferenceTypeGroupcall),
// 				"reference_id":   "15592456-2a5b-11ef-8a97-27d50d25a280",
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
// 				mockResource.EXPECT().UpdateData(ctx, r.ID, tt.groupcall).Return(&resource.Resource{}, nil)
// 			}

// 			if err := h.webhookGroupcallUpdated(ctx, tt.groupcall); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 		})
// 	}
// }
