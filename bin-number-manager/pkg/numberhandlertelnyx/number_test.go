package numberhandlertelnyx

// func Test_CreateNumber(t *testing.T) {

// 	type test struct {
// 		name string

// 		number string

// 		responseOrder  *telnyx.OrderNumber
// 		responseNumber *telnyx.PhoneNumber

// 		expectRes *providernumber.ProviderNumber
// 	}

// 	tests := []test{
// 		{
// 			"normal",

// 			"+821021656521",

// 			&telnyx.OrderNumber{
// 				PhoneNumbers: []telnyx.OrderNumberPhoneNumber{
// 					{
// 						ID:          "1748688147379652251",
// 						PhoneNumber: "+821021656521",
// 						Status:      "active",
// 					},
// 				},
// 			},
// 			&telnyx.PhoneNumber{
// 				ID:                    "1748688147379652251",
// 				RecordType:            "phone_number",
// 				PhoneNumber:           "+12704940136",
// 				Status:                telnyx.PhoneNumberStatusActive,
// 				Tags:                  []string{},
// 				ConnectionID:          "tmp connection id",
// 				T38FaxGatewayEnabled:  true,
// 				PurchasedAt:           "2021-02-26T18:26:49Z",
// 				EmergencyEnabled:      false,
// 				CallForwardingEnabled: true,
// 				CNAMListingEnabled:    false,
// 				CallRecordingEnabled:  false,
// 				CreatedAt:             "2021-02-26T18:26:49.277Z",
// 				UpdatedAt:             "2021-02-27T17:07:16.234Z",
// 			},

// 			&providernumber.ProviderNumber{
// 				ID:               "1748688147379652251",
// 				Status:           number.StatusActive,
// 				T38Enabled:       true,
// 				EmergencyEnabled: false,
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockExternal := requestexternal.NewMockRequestExternal(mc)

// 			h := numberHandlerTelnyx{
// 				reqHandler:      mockReq,
// 				db:              mockDB,
// 				requestExternal: mockExternal,
// 			}

// 			numbers := []string{tt.number}
// 			mockExternal.EXPECT().TelnyxNumberOrdersPost(defaultToken, numbers, defaultConnectionID, defaultMessagingProfileID).Return(tt.responseOrder, nil)
// 			mockExternal.EXPECT().TelnyxPhoneNumbersGetByNumber(defaultToken, tt.number).Return(tt.responseNumber, nil)
// 			res, err := h.NumberPurchase(tt.number)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			if !reflect.DeepEqual(tt.expectRes, res) {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
// 			}
// 		})
// 	}
// }

// func Test_NumberRelease(t *testing.T) {

// 	type test struct {
// 		name   string
// 		number *number.Number
// 	}

// 	tests := []test{
// 		{
// 			"normal",
// 			&number.Number{
// 				ID:                  uuid.FromStringOrNil("d8659476-79e1-11eb-a59b-9301c8a84847"),
// 				ProviderReferenceID: "1580568175064384684",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockExternal := requestexternal.NewMockRequestExternal(mc)

// 			h := numberHandlerTelnyx{
// 				reqHandler:      mockReq,
// 				db:              mockDB,
// 				requestExternal: mockExternal,
// 			}

// 			ctx := context.Background()

// 			mockExternal.EXPECT().TelnyxPhoneNumbersIDDelete(defaultToken, tt.number.ProviderReferenceID)
// 			// mockDB.EXPECT().NumberDelete(gomock.Any(), tt.number.ID)
// 			// mockDB.EXPECT().NumberGet(gomock.Any(), tt.number.ID)
// 			if err := h.NumberRelease(ctx, tt.number); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 		})
// 	}
// }
