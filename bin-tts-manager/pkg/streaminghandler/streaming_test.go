package streaminghandler

// func Test_Create(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		customerID   uuid.UUID
// 		transcribeID uuid.UUID
// 		language     string
// 		direction    transcript.Direction

// 		responseID uuid.UUID

// 		expectRes *streaming.Streaming
// 	}{
// 		{
// 			name: "normal",

// 			customerID:   uuid.FromStringOrNil("b3755cc4-e9da-11ef-812f-a73170810307"),
// 			transcribeID: uuid.FromStringOrNil("b3d7ba72-e9da-11ef-8646-f320e52dfd72"),
// 			language:     "en-US",
// 			direction:    transcript.DirectionIn,

// 			responseID: uuid.FromStringOrNil("b422ebbe-e9da-11ef-8537-332e8845ed9e"),

// 			expectRes: &streaming.Streaming{
// 				Identity: commonidentity.Identity{
// 					ID:         uuid.FromStringOrNil("b422ebbe-e9da-11ef-8537-332e8845ed9e"),
// 					CustomerID: uuid.FromStringOrNil("b3755cc4-e9da-11ef-812f-a73170810307"),
// 				},
// 				TranscribeID: uuid.FromStringOrNil("b3d7ba72-e9da-11ef-8646-f320e52dfd72"),
// 				Language:     "en-US",
// 				Direction:    transcript.DirectionIn,
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		tt := tt
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockUtil := utilhandler.NewMockUtilHandler(mc)
// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockNotfiy := notifyhandler.NewMockNotifyHandler(mc)

// 			h := &streamingHandler{
// 				utilHandler:   mockUtil,
// 				reqHandler:    mockReq,
// 				notifyHandler: mockNotfiy,
// 				mapStreaming:  make(map[uuid.UUID]*streaming.Streaming),
// 			}
// 			ctx := context.Background()

// 			mockUtil.EXPECT().UUIDCreate().Return(tt.responseID)
// 			mockNotfiy.EXPECT().PublishEvent(ctx, streaming.EventTypeStreamingStarted, tt.expectRes)

// 			res, err := h.Create(ctx, tt.customerID, tt.transcribeID, tt.language, tt.direction)
// 			if err != nil {
// 				t.Errorf("Wrong match. expected: ok, got: %v", err)
// 			}

// 			if !reflect.DeepEqual(res, tt.expectRes) {
// 				t.Errorf("Wrong match. expected: %v, got: %v", tt.expectRes, res)
// 			}

// 			resGet, err := h.Get(ctx, tt.responseID)
// 			if err != nil {
// 				t.Errorf("Wrong match. expected: ok, got: %v", err)
// 			}

// 			if !reflect.DeepEqual(resGet, tt.expectRes) {
// 				t.Errorf("Wrong match. expected: %v, got: %v", tt.expectRes, resGet)
// 			}

// 			mockNotfiy.EXPECT().PublishEvent(ctx, streaming.EventTypeStreamingStopped, tt.expectRes)
// 			h.Delete(ctx, tt.responseID)
// 			resDelete, err := h.Get(ctx, tt.responseID)
// 			if err == nil {
// 				t.Errorf("Wrong match. expected: error, got: ok")
// 			}

// 			if resDelete != nil {
// 				t.Errorf("Wrong match. expected: nil, got: %v", resDelete)
// 			}

// 		})
// 	}
// }
